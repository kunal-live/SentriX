#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <ctype.h>
#include <sys/stat.h>
#ifndef _WIN32
#include <pwd.h>
#include <grp.h>
#endif
#include "collector.h"
#include "checks.h"
#include "buffer.h"

extern void build_telemetry_json(char *buffer, size_t size, const char *agent_id, uint64_t seq, const SystemMetrics *m);
extern int send_telemetry(const char *host, int port, const char *path, const char *json_payload, const char *auth_token);

#define CONFIG_MAX_LEN 256

typedef struct {
    char server_host[CONFIG_MAX_LEN];
    int server_port;
    char agent_id[CONFIG_MAX_LEN];
    char auth_token[CONFIG_MAX_LEN];
    int use_tls;
} AgentConfig;

static void trim_inplace(char *str) {
    if (!str) return;
    char *end;
    while (isspace((unsigned char)*str)) str++;
    if (*str == 0) return;
    end = str + strlen(str) - 1;
    while (end > str && isspace((unsigned char)*end)) end--;
    end[1] = '\0';
}

static void load_config_file(const char *filepath, AgentConfig *cfg) {
    struct stat st;
    if (stat(filepath, &st) == 0) {
        if ((st.st_mode & 0077) != 0) {
            fprintf(stderr, "SECURITY WARNING: %s has overly permissive permissions! (Recommended: chmod 0600)\n", filepath);
        }
    }

    FILE *fp = fopen(filepath, "r");
    if (!fp) return;

    char line[512];
    while (fgets(line, sizeof(line), fp)) {
        // Strip comments and empty lines
        char *p = strchr(line, '#');
        if (p) *p = '\0';
        trim_inplace(line);
        if (line[0] == '\0') continue;

        char *eq = strchr(line, '=');
        if (!eq) continue;

        *eq = '\0';
        char *key = line;
        char *val = eq + 1;
        trim_inplace(key);
        trim_inplace(val);

        // Remove quotes if present
        if ((val[0] == '"' || val[0] == '\'') && val[strlen(val) - 1] == val[0]) {
            val[strlen(val) - 1] = '\0';
            val++;
        }

        if (strcmp(key, "SENTRIX_SERVER_HOST") == 0 && val[0] != '\0') {
            strncpy(cfg->server_host, val, sizeof(cfg->server_host) - 1);
        } else if (strcmp(key, "SENTRIX_SERVER_PORT") == 0 && val[0] != '\0') {
            cfg->server_port = atoi(val);
        } else if (strcmp(key, "SENTRIX_AGENT_ID") == 0 && val[0] != '\0') {
            strncpy(cfg->agent_id, val, sizeof(cfg->agent_id) - 1);
        } else if (strcmp(key, "SENTRIX_AGENT_CREDENTIAL") == 0 && val[0] != '\0') {
            strncpy(cfg->auth_token, val, sizeof(cfg->auth_token) - 1);
        } else if (strcmp(key, "SENTRIX_USE_TLS") == 0 && val[0] != '\0') {
            cfg->use_tls = (strcmp(val, "1") == 0 || strcasecmp(val, "true") == 0);
        }
    }

    fclose(fp);
    printf("Loaded configuration from %s\n", filepath);
}

static void load_env_vars(AgentConfig *cfg) {
    const char *env_host = getenv("SENTRIX_SERVER_HOST");
    if (env_host && env_host[0] != '\0') {
        strncpy(cfg->server_host, env_host, sizeof(cfg->server_host) - 1);
    }

    const char *env_port = getenv("SENTRIX_SERVER_PORT");
    if (env_port && env_port[0] != '\0') {
        cfg->server_port = atoi(env_port);
    }

    const char *env_id = getenv("SENTRIX_AGENT_ID");
    if (env_id && env_id[0] != '\0') {
        strncpy(cfg->agent_id, env_id, sizeof(cfg->agent_id) - 1);
    }

    const char *env_cred = getenv("SENTRIX_AGENT_CREDENTIAL");
    if (env_cred && env_cred[0] != '\0') {
        strncpy(cfg->auth_token, env_cred, sizeof(cfg->auth_token) - 1);
    }

    const char *env_tls = getenv("SENTRIX_USE_TLS");
    if (env_tls && env_tls[0] != '\0') {
        cfg->use_tls = (strcmp(env_tls, "1") == 0 || strcasecmp(env_tls, "true") == 0);
    }
}

static void maybe_drop_privileges(void) {
#ifndef _WIN32
    if (getuid() == 0) {
        struct passwd *pw = getpwnam("sentrix");
        if (!pw) {
            pw = getpwnam("nobody");
        }
        if (pw) {
            if (setgid(pw->pw_gid) == 0 && setuid(pw->pw_uid) == 0) {
                printf("Security: Dropped root privileges to user '%s' (UID %d, GID %d)\n", pw->pw_name, pw->pw_uid, pw->pw_gid);
            }
        }
    }
#endif
}

static void print_usage(const char *prog) {
    printf("Usage: %s [OPTIONS]\n", prog);
    printf("Options:\n");
    printf("  -c <file>      Path to config file (default: /etc/sentrix/agent.env)\n");
    printf("  -h <host>      SentriX server host\n");
    printf("  -p <port>      SentriX server port (default: 8080)\n");
    printf("  -s, --tls      Enforce TLS/HTTPS transport (default on port 443)\n");
    printf("  -i <agent_id>  Agent UUID\n");
    printf("  --help         Show this help message\n");
}

int main(int argc, char *argv[]) {
    printf("SentriX Agent v1.0.0 starting...\n");

    AgentConfig config;
    memset(&config, 0, sizeof(config));

    // Default values (fallback)
    strncpy(config.server_host, "127.0.0.1", sizeof(config.server_host) - 1);
    config.server_port = 8080;
    config.agent_id[0] = '\0';
    config.auth_token[0] = '\0';

    const char *config_file = "/etc/sentrix/agent.env";

    // 1. Scan command line for custom config file first
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "-c") == 0 && i + 1 < argc) {
            config_file = argv[++i];
        } else if (strcmp(argv[i], "--help") == 0) {
            print_usage(argv[0]);
            return 0;
        }
    }

    // 2. Load from file if accessible
    load_config_file(config_file, &config);

    // 3. Override from environment variables
    load_env_vars(&config);

    // 4. Override from CLI flags
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "-h") == 0 && i + 1 < argc) {
            strncpy(config.server_host, argv[++i], sizeof(config.server_host) - 1);
        } else if (strcmp(argv[i], "-p") == 0 && i + 1 < argc) {
            config.server_port = atoi(argv[++i]);
        } else if (strcmp(argv[i], "-i") == 0 && i + 1 < argc) {
            strncpy(config.agent_id, argv[++i], sizeof(config.agent_id) - 1);
        } else if (strcmp(argv[i], "-s") == 0 || strcmp(argv[i], "--tls") == 0) {
            config.use_tls = 1;
        } else if (strcmp(argv[i], "-t") == 0) {
            fprintf(stderr, "SECURITY ERROR: Passing credential tokens via -t CLI arguments is disabled to prevent exposure in process lists (/proc/<pid>/cmdline). Configure SENTRIX_AGENT_CREDENTIAL in %s instead.\n", config_file);
            return 1;
        }
    }

    if (config.server_port == 443 || config.server_port == 8443 || config.use_tls) {
        config.use_tls = 1;
#ifndef _WIN32
        setenv("SENTRIX_USE_TLS", "true", 1);
#else
        _putenv("SENTRIX_USE_TLS=true");
#endif
    }

    if (config.auth_token[0] == '\0') {
        fprintf(stderr, "FATAL CONFIGURATION ERROR: No agent authentication credential found in %s or SENTRIX_AGENT_CREDENTIAL. Cannot start daemon without authentication credentials.\n", config_file);
        return 1;
    }

    if (config.agent_id[0] == '\0') {
        fprintf(stderr, "FATAL CONFIGURATION ERROR: No agent ID found in %s or SENTRIX_AGENT_ID. Cannot start daemon without a valid agent ID.\n", config_file);
        return 1;
    }

    printf("Target Server: %s:%d (TLS: %s)\n", config.server_host, config.server_port, config.use_tls ? "ENABLED" : "DISABLED");
    printf("Agent ID:      %s\n", config.agent_id);

    maybe_drop_privileges();

    DiskBuffer spool_buf;
    const char *spool_path = getenv("SENTRIX_SPOOL_PATH");
    if (!spool_path) {
#ifndef _WIN32
        // Prefer /var/lib/sentrix/sentrix_spool.bin if the directory exists and is writable
        if (access("/var/lib/sentrix", W_OK) == 0) {
            spool_path = "/var/lib/sentrix/sentrix_spool.bin";
        } else {
            spool_path = "sentrix_spool.bin";
        }
#else
        spool_path = "sentrix_spool.bin";
#endif
    }
    if (buffer_init(&spool_buf, spool_path, 0) == 0) {
        printf("Initialized local disk spool buffer at %s (%d records queued).\n", spool_buf.file_path, spool_buf.items_queued);
    } else {
        fprintf(stderr, "WARNING: Failed to initialize disk spool buffer at %s. Offline telemetry disk caching disabled.\n", spool_path);
    }

    uint64_t sequence = 1;
    char json_buffer[2048];
    SystemMetrics metrics;

    typedef enum {
        AGENT_STATE_ONLINE = 0,
        AGENT_STATE_SPOOLING = 1,
        AGENT_STATE_DRAINING = 2
    } AgentTransportState;
    AgentTransportState transport_state = (spool_buf.items_queued > 0) ? AGENT_STATE_DRAINING : AGENT_STATE_ONLINE;

    while (1) {
        printf("Collecting metrics (seq %lu)...\n", (unsigned long)sequence);

        collect_cpu(&metrics.cpu_utilization);
        collect_memory(&metrics.mem_total_bytes, &metrics.mem_used_bytes);
        collect_disk(&metrics.disk_total_bytes, &metrics.disk_used_bytes);
        collect_network(&metrics.net_rx_bytes, &metrics.net_tx_bytes);
        collect_system_load(&metrics.load_avg_1m);
        collect_uptime(&metrics.uptime_seconds);

        build_telemetry_json(json_buffer, sizeof(json_buffer), config.agent_id, sequence, &metrics);

        // 1. Drain offline spool buffer if connected
        if (spool_buf.items_queued > 0 && transport_state != AGENT_STATE_DRAINING) {
            transport_state = AGENT_STATE_DRAINING;
            printf("[State] Transition -> DRAINING_BUFFER (%d queued frames)\n", spool_buf.items_queued);
        }
        while (spool_buf.items_queued > 0) {
            char spooled_json[4096];
            size_t spooled_len = 0;
            if (buffer_peek(&spool_buf, spooled_json, sizeof(spooled_json), &spooled_len) == 0) {
                if (send_telemetry(config.server_host, config.server_port, "/api/v1/agent/telemetry", spooled_json, config.auth_token) == 0) {
                    printf("[Spool] Drained queued metric frame (%d remaining)\n", spool_buf.items_queued - 1);
                    buffer_pop(&spool_buf);
                } else {
                    if (transport_state != AGENT_STATE_SPOOLING) {
                        transport_state = AGENT_STATE_SPOOLING;
                        printf("[State] Transition -> OFFLINE_SPOOLING\n");
                    }
                    printf("[Spool] Connection not ready to drain queue. Retrying next cycle.\n");
                    break;
                }
            } else {
                buffer_pop(&spool_buf); // corrupted record, discard
            }
        }

        // 2. Transmit current telemetry frame or buffer on network partition
        if (send_telemetry(config.server_host, config.server_port, "/api/v1/agent/telemetry", json_buffer, config.auth_token) == 0) {
            if (transport_state != AGENT_STATE_ONLINE) {
                transport_state = AGENT_STATE_ONLINE;
                printf("[State] Transition -> ONLINE_STREAMING\n");
            }
            printf("Telemetry sent successfully.\n");
        } else {
            if (transport_state != AGENT_STATE_SPOOLING) {
                transport_state = AGENT_STATE_SPOOLING;
                printf("[State] Transition -> OFFLINE_SPOOLING\n");
            }
            printf("Telemetry dispatch failed. Spooling frame to local buffer...\n");
            if (buffer_push(&spool_buf, json_buffer, strlen(json_buffer)) == 0) {
                printf("[Spool] Frame spooled successfully (%d items in queue, %zu bytes).\n",
                       spool_buf.items_queued, spool_buf.current_bytes);
            } else {
                printf("[Spool] Warning: Failed to buffer telemetry to disk.\n");
            }
        }

        // Execute synthetic checks cycle every 3 intervals (30 seconds)
        if (sequence % 3 == 0) {
            printf("Running synthetic checks cycle...\n");
            run_checks_cycle(config.server_host, config.server_port, config.agent_id, config.auth_token);
        }

        sequence++;
        sleep(10); // 10 second telemetry cycle
    }

    return 0;
}
