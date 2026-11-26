#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include "collector.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <ctype.h>
#include <unistd.h>
#include <sys/statvfs.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <pwd.h>
#include <time.h>

int collect_disk(uint64_t *total, uint64_t *used) {
    struct statvfs stat;
    if (statvfs("/", &stat) != 0) return -1;
    
    *total = (uint64_t)stat.f_blocks * stat.f_frsize;
    uint64_t free_space = (uint64_t)stat.f_bfree * stat.f_frsize;
    *used = *total - free_space;
    return 0;
}

int collect_network(uint64_t *rx, uint64_t *tx) {
    FILE *fp = fopen("/proc/net/dev", "r");
    if (!fp) return -1;

    char line[512];
    uint64_t total_rx = 0;
    uint64_t total_tx = 0;

    // Skip the first two header lines
    if (!fgets(line, sizeof(line), fp)) { fclose(fp); return -1; }
    if (!fgets(line, sizeof(line), fp)) { fclose(fp); return -1; }

    while (fgets(line, sizeof(line), fp)) {
        char *colon = strchr(line, ':');
        if (!colon) continue;

        *colon = '\0';
        char *iface = line;
        while (*iface == ' ') iface++;

        // Ignore loopback interface
        if (strcmp(iface, "lo") == 0) continue;

        uint64_t r_bytes = 0, t_bytes = 0;
        if (sscanf(colon + 1, "%lu %*u %*u %*u %*u %*u %*u %*u %lu", &r_bytes, &t_bytes) >= 2) {
            total_rx += r_bytes;
            total_tx += t_bytes;
        }
    }

    fclose(fp);
    *rx = total_rx;
    *tx = total_tx;
    return 0;
}

int collect_system_load(double *load_1m) {
    if (!load_1m) return -1;
    *load_1m = 0.0;
    FILE *fp = fopen("/proc/loadavg", "r");
    if (!fp) return -1;
    if (fscanf(fp, "%lf", load_1m) != 1) {
        *load_1m = 0.0;
    }
    fclose(fp);
    return 0;
}

int collect_uptime(uint64_t *seconds) {
    if (!seconds) return -1;
    *seconds = 0;
    FILE *fp = fopen("/proc/uptime", "r");
    if (!fp) return -1;
    double up = 0.0;
    if (fscanf(fp, "%lf", &up) == 1) {
        *seconds = (uint64_t)up;
    }
    fclose(fp);
    return 0;
}

typedef struct {
    int pid;
    unsigned long long ticks; /* utime + stime */
} ProcSample;

typedef struct {
    ProcessInfo info;
    unsigned long long ticks;
} ProcCandidate;

/* Previous sample, sorted by pid, used to turn CPU ticks into a percentage. */
static ProcSample *g_prev = NULL;
static int g_prev_count = 0;
static double g_prev_time = 0.0;

static double monotonic_seconds(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (double)ts.tv_sec + (double)ts.tv_nsec / 1e9;
}

static int cmp_sample_pid(const void *a, const void *b) {
    const ProcSample *x = a, *y = b;
    return (x->pid > y->pid) - (x->pid < y->pid);
}

static int cmp_candidate_busy(const void *a, const void *b) {
    const ProcCandidate *x = a, *y = b;
    if (x->info.cpu_percent != y->info.cpu_percent) {
        return x->info.cpu_percent < y->info.cpu_percent ? 1 : -1;
    }
    return (x->info.mem_rss_bytes < y->info.mem_rss_bytes) - (x->info.mem_rss_bytes > y->info.mem_rss_bytes);
}

static const char *state_name(char c) {
    switch (c) {
        case 'R': return "RUNNING";
        case 'S': return "SLEEPING";
        case 'D': return "DISK_WAIT";
        case 'Z': return "ZOMBIE";
        case 'T': case 't': return "STOPPED";
        case 'I': return "IDLE";
        default:  return "OTHER";
    }
}

static void uid_to_name(uid_t uid, char *out, size_t out_size) {
    struct passwd pwd, *res = NULL;
    char buf[1024];
    if (getpwuid_r(uid, &pwd, buf, sizeof(buf), &res) == 0 && res) {
        snprintf(out, out_size, "%s", res->pw_name);
    } else {
        snprintf(out, out_size, "%u", (unsigned)uid);
    }
}

/* Parses /proc/<pid>/stat. Returns 0 on success. */
static int read_proc_stat(int pid, char *name, size_t name_size, char *state,
                          unsigned long long *ticks, long *rss_pages) {
    char path[64];
    snprintf(path, sizeof(path), "/proc/%d/stat", pid);
    FILE *fp = fopen(path, "r");
    if (!fp) return -1;
    char line[1024];
    char *ok = fgets(line, sizeof(line), fp);
    fclose(fp);
    if (!ok) return -1;

    /* The command name is in parentheses and may itself contain spaces or ')',
     * so split on the first '(' and the last ')'. */
    char *open_p = strchr(line, '(');
    char *close_p = strrchr(line, ')');
    if (!open_p || !close_p || close_p < open_p) return -1;
    size_t len = (size_t)(close_p - open_p - 1);
    if (len >= name_size) len = name_size - 1;
    memcpy(name, open_p + 1, len);
    name[len] = '\0';

    unsigned long long utime = 0, stime = 0;
    long rss = 0;
    /* Fields after the name: state(3) ... utime(14) stime(15) ... rss(24). */
    int n = sscanf(close_p + 1,
                   " %c %*d %*d %*d %*d %*d %*u %*u %*u %*u %*u %llu %llu"
                   " %*d %*d %*d %*d %*d %*d %*u %*u %ld",
                   state, &utime, &stime, &rss);
    if (n != 4) return -1;
    *ticks = utime + stime;
    *rss_pages = rss;
    return 0;
}

int collect_top_processes(ProcessInfo *procs, int max_count, int *out_count, int *total_count) {
    *out_count = 0;
    if (total_count) *total_count = 0;

    DIR *dir = opendir("/proc");
    if (!dir) return -1;

    long clk_tck = sysconf(_SC_CLK_TCK);
    long page_size = sysconf(_SC_PAGESIZE);
    if (clk_tck <= 0) clk_tck = 100;
    if (page_size <= 0) page_size = 4096;

    double now = monotonic_seconds();
    double elapsed = (g_prev_time > 0.0) ? now - g_prev_time : 0.0;

    int cap = 512, count = 0;
    ProcCandidate *cands = malloc(sizeof(ProcCandidate) * (size_t)cap);
    if (!cands) {
        closedir(dir);
        return -1;
    }

    struct dirent *entry;
    while ((entry = readdir(dir)) != NULL) {
        if (!isdigit((unsigned char)entry->d_name[0])) continue;
        int pid = atoi(entry->d_name);

        char name[64];
        char state = '?';
        unsigned long long ticks = 0;
        long rss_pages = 0;
        if (read_proc_stat(pid, name, sizeof(name), &state, &ticks, &rss_pages) != 0) {
            continue; /* exited while scanning, or unreadable */
        }

        if (count == cap) {
            ProcCandidate *grown = realloc(cands, sizeof(ProcCandidate) * (size_t)cap * 2);
            if (!grown) break;
            cands = grown;
            cap *= 2;
        }

        ProcCandidate *c = &cands[count++];
        memset(c, 0, sizeof(*c));
        c->info.pid = pid;
        c->ticks = ticks;
        snprintf(c->info.name, sizeof(c->info.name), "%s", name);
        snprintf(c->info.state, sizeof(c->info.state), "%s", state_name(state));
        c->info.mem_rss_bytes = rss_pages > 0 ? (uint64_t)rss_pages * (uint64_t)page_size : 0;

        if (elapsed > 0.0 && g_prev) {
            ProcSample key = { pid, 0 };
            ProcSample *prev = bsearch(&key, g_prev, (size_t)g_prev_count, sizeof(ProcSample), cmp_sample_pid);
            if (prev && ticks >= prev->ticks) {
                c->info.cpu_percent = 100.0 * ((double)(ticks - prev->ticks) / (double)clk_tck) / elapsed;
            }
        }

        char path[64];
        struct stat st;
        snprintf(path, sizeof(path), "/proc/%d", pid);
        if (stat(path, &st) == 0) {
            uid_to_name(st.st_uid, c->info.user, sizeof(c->info.user));
        } else {
            snprintf(c->info.user, sizeof(c->info.user), "?");
        }
    }
    closedir(dir);

    /* Remember this sample for the next call. */
    ProcSample *next = malloc(sizeof(ProcSample) * (size_t)(count > 0 ? count : 1));
    if (next) {
        for (int i = 0; i < count; i++) {
            next[i].pid = cands[i].info.pid;
            next[i].ticks = cands[i].ticks;
        }
        qsort(next, (size_t)count, sizeof(ProcSample), cmp_sample_pid);
        free(g_prev);
        g_prev = next;
        g_prev_count = count;
        g_prev_time = now;
    }

    qsort(cands, (size_t)count, sizeof(ProcCandidate), cmp_candidate_busy);
    int n = count < max_count ? count : max_count;
    for (int i = 0; i < n; i++) {
        procs[i] = cands[i].info;
    }
    free(cands);

    *out_count = n;
    if (total_count) *total_count = count;
    return 0;
}
