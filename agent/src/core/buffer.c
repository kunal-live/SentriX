#include "buffer.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <errno.h>
#include <sys/stat.h>

#ifndef _WIN32
#include <fcntl.h>
#include <unistd.h>
#endif

int buffer_init(DiskBuffer *buf, const char *path, size_t max_bytes) {
    if (!buf) return -1;
    strncpy(buf->file_path, path ? path : DEFAULT_BUFFER_FILE, sizeof(buf->file_path) - 1);
    buf->file_path[sizeof(buf->file_path) - 1] = '\0';
    buf->max_bytes = max_bytes > 0 ? max_bytes : DEFAULT_MAX_BUFFER_BYTES;
    buf->current_bytes = 0;
    buf->items_queued = 0;

    // Reject symbolic links when opening existing spool file
#ifndef _WIN32
    int fd = open(buf->file_path, O_RDONLY | O_NOFOLLOW);
    if (fd < 0) {
        if (errno == ENOENT) {
            // File does not exist yet; normal startup
            return 0;
        }
        fprintf(stderr, "buffer_init: failed to open spool %s (errno %d: %s)\n",
                buf->file_path, errno, strerror(errno));
        return -1;
    }
    FILE *fp = fdopen(fd, "rb");
    if (!fp) {
        close(fd);
        return -1;
    }
#else
    FILE *fp = fopen(buf->file_path, "rb");
    if (!fp) {
        if (errno == ENOENT) return 0;
        return -1;
    }
#endif

    uint32_t record_len = 0;
    while (fread(&record_len, sizeof(uint32_t), 1, fp) == 1) {
        // Sanity check length to prevent corrupted buffer loop
        if (record_len > 10 * 1024 * 1024) {
            fprintf(stderr, "buffer_init: detected corrupted spool record length (%u bytes)\n", record_len);
            fclose(fp);
            return -1;
        }
        if (fseek(fp, (long)record_len, SEEK_CUR) != 0) {
            break;
        }
        buf->items_queued++;
        buf->current_bytes += sizeof(uint32_t) + record_len;
    }

    if (fclose(fp) != 0) {
        return -1;
    }
    return 0;
}

int buffer_push(DiskBuffer *buf, const char *payload, size_t length) {
    if (!buf || !payload || length == 0) return -1;

    // Validate length for overflow and maximum payload size (10 MB cap)
    if (length > (size_t)(UINT32_MAX - sizeof(uint32_t)) || length > 10 * 1024 * 1024) {
        return -1;
    }

    // Check max buffer limit
    while (buf->items_queued > 0 && buf->current_bytes + sizeof(uint32_t) + length > buf->max_bytes) {
        // Drop oldest record by popping
        if (buffer_pop(buf) != 0) {
            break;
        }
    }

    // Open with atomic 0600 mode and O_NOFOLLOW symlink protection
#ifndef _WIN32
    int fd = open(buf->file_path, O_WRONLY | O_CREAT | O_APPEND | O_NOFOLLOW, 0600);
    if (fd < 0) {
        fprintf(stderr, "buffer_push: open failed on %s (errno %d: %s)\n",
                buf->file_path, errno, strerror(errno));
        return -1;
    }
    FILE *fp = fdopen(fd, "ab");
    if (!fp) {
        close(fd);
        return -1;
    }
#else
    FILE *fp = fopen(buf->file_path, "ab");
    if (!fp) return -1;
#endif

    uint32_t len32 = (uint32_t)length;
    if (fwrite(&len32, sizeof(uint32_t), 1, fp) != 1) {
        fclose(fp);
        return -1;
    }
    if (fwrite(payload, 1, length, fp) != length) {
        fclose(fp);
        return -1;
    }
    if (fclose(fp) != 0) {
        return -1;
    }

    buf->items_queued++;
    buf->current_bytes += sizeof(uint32_t) + length;
    return 0;
}

int buffer_peek(DiskBuffer *buf, char *out_payload, size_t max_out_len, size_t *out_len) {
    if (!buf || !out_payload || buf->items_queued == 0) return -1;

#ifndef _WIN32
    int fd = open(buf->file_path, O_RDONLY | O_NOFOLLOW);
    if (fd < 0) return -1;
    FILE *fp = fdopen(fd, "rb");
    if (!fp) {
        close(fd);
        return -1;
    }
#else
    FILE *fp = fopen(buf->file_path, "rb");
    if (!fp) return -1;
#endif

    uint32_t len32 = 0;
    if (fread(&len32, sizeof(uint32_t), 1, fp) != 1) {
        fclose(fp);
        return -3; // Truncated/corrupt record header
    }

    if (len32 >= max_out_len) {
        fclose(fp);
        return -2; // Record larger than caller buffer: can never be sent
    }

    size_t read_bytes = fread(out_payload, 1, len32, fp);
    if (fclose(fp) != 0) {
        return -1;
    }

    if (read_bytes != len32) return -3; // Truncated/corrupt record body
    out_payload[read_bytes] = '\0';
    if (out_len) *out_len = read_bytes;

    return 0;
}

int buffer_pop(DiskBuffer *buf) {
    if (!buf || buf->items_queued == 0) return -1;

#ifndef _WIN32
    int in_fd = open(buf->file_path, O_RDONLY | O_NOFOLLOW);
    if (in_fd < 0) return -1;
    FILE *fp = fdopen(in_fd, "rb");
    if (!fp) {
        close(in_fd);
        return -1;
    }
#else
    FILE *fp = fopen(buf->file_path, "rb");
    if (!fp) return -1;
#endif

    uint32_t len32 = 0;
    if (fread(&len32, sizeof(uint32_t), 1, fp) != 1) {
        fclose(fp);
        return -1;
    }

    // Skip the first record and read the rest into temp file
    char temp_path[300];
    snprintf(temp_path, sizeof(temp_path), "%s.tmp", buf->file_path);

#ifndef _WIN32
    int out_fd = open(temp_path, O_WRONLY | O_CREAT | O_TRUNC | O_NOFOLLOW, 0600);
    if (out_fd < 0) {
        fclose(fp);
        return -1;
    }
    FILE *out = fdopen(out_fd, "wb");
    if (!out) {
        close(out_fd);
        fclose(fp);
        return -1;
    }
#else
    FILE *out = fopen(temp_path, "wb");
    if (!out) {
        fclose(fp);
        return -1;
    }
#endif

    if (fseek(fp, (long)len32, SEEK_CUR) != 0) {
        fclose(fp);
        fclose(out);
        remove(temp_path);
        return -1;
    }

    char copy_buf[4096];
    size_t n;
    size_t remaining_bytes = 0;
    while ((n = fread(copy_buf, 1, sizeof(copy_buf), fp)) > 0) {
        if (fwrite(copy_buf, 1, n, out) != n) {
            fclose(fp);
            fclose(out);
            remove(temp_path);
            return -1;
        }
        remaining_bytes += n;
    }

    if (ferror(fp)) {
        fclose(fp);
        fclose(out);
        remove(temp_path);
        return -1;
    }

    if (fclose(fp) != 0 || fclose(out) != 0) {
        remove(temp_path);
        return -1;
    }

#ifdef _WIN32
    // Windows rename() cannot replace an existing file. On POSIX, rename() is
    // atomic, so removing first would open a window where a crash loses the
    // whole spool.
    remove(buf->file_path);
#endif
    if (rename(temp_path, buf->file_path) != 0) {
        remove(temp_path);
        return -1;
    }

    buf->items_queued--;
    buf->current_bytes = remaining_bytes;
    return 0;
}

int buffer_size(const DiskBuffer *buf) {
    return buf ? buf->items_queued : 0;
}

void buffer_cleanup(DiskBuffer *buf) {
    if (buf) {
        remove(buf->file_path);
        buf->items_queued = 0;
        buf->current_bytes = 0;
    }
}
