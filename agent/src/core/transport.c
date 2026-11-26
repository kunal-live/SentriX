#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include "transport.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <netdb.h>
#include <time.h>
#include <fcntl.h>
#include <errno.h>
#include "collector.h"
#include "cJSON.h"

#ifdef SENTRIX_ENABLE_TLS
#include <openssl/ssl.h>
#include <openssl/err.h>
#include <openssl/x509v3.h>

static SSL_CTX *g_ssl_ctx = NULL;

static SSL_CTX *get_ssl_context(void) {
    if (g_ssl_ctx) {
        return g_ssl_ctx;
    }

    SSL_library_init();
    SSL_load_error_strings();
    OpenSSL_add_all_algorithms();

    const SSL_METHOD *method = TLS_client_method();
    g_ssl_ctx = SSL_CTX_new(method);
    if (!g_ssl_ctx) {
        fprintf(stderr, "TLS Error: Failed to initialize OpenSSL context\n");
        return NULL;
    }

    // Enforce TLS 1.2 minimum protocol version
#if defined(TLS1_2_VERSION)
    SSL_CTX_set_min_proto_version(g_ssl_ctx, TLS1_2_VERSION);
#endif

    // Load default trusted system CA bundles
    if (SSL_CTX_set_default_verify_paths(g_ssl_ctx) != 1) {
        fprintf(stderr, "TLS Warning: Could not load default system CA store\n");
    }

    // Support custom enterprise CA certificate path
    const char *custom_ca = getenv("SENTRIX_CA_CERT");
    if (!custom_ca) {
        if (access("/etc/sentrix/ca.crt", R_OK) == 0) {
            custom_ca = "/etc/sentrix/ca.crt";
        }
    }

    if (custom_ca && access(custom_ca, R_OK) == 0) {
        if (SSL_CTX_load_verify_locations(g_ssl_ctx, custom_ca, NULL) == 1) {
            printf("TLS: Loaded custom CA certificate from %s\n", custom_ca);
        } else {
            fprintf(stderr, "TLS Warning: Failed to load CA from %s\n", custom_ca);
        }
    }

#ifdef SENTRIX_ALLOW_INSECURE_TLS
    const char *skip_verify = getenv("SENTRIX_INSECURE_SKIP_VERIFY");
    if (skip_verify && (strcmp(skip_verify, "1") == 0 || strcmp(skip_verify, "true") == 0)) {
        fprintf(stderr, "SECURITY WARNING: SENTRIX_INSECURE_SKIP_VERIFY is active! Server certificate will NOT be verified.\n");
        SSL_CTX_set_verify(g_ssl_ctx, SSL_VERIFY_NONE, NULL);
    } else {
        SSL_CTX_set_verify(g_ssl_ctx, SSL_VERIFY_PEER, NULL);
    }
#else
    SSL_CTX_set_verify(g_ssl_ctx, SSL_VERIFY_PEER, NULL);
#endif

    return g_ssl_ctx;
}
#endif

void build_telemetry_json(char *buffer, size_t size, const char *agent_id, uint64_t seq, const char *restart_nonce, const SystemMetrics *m) {
    if (!buffer || size == 0) return;

    time_t now = time(NULL);
    struct tm *tm_info = gmtime(&now);
    char timestamp[32];
    strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%SZ", tm_info);

    cJSON *root = cJSON_CreateObject();
    cJSON_AddStringToObject(root, "agent_id", agent_id ? agent_id : "");
    cJSON_AddStringToObject(root, "timestamp", timestamp);
    cJSON_AddNumberToObject(root, "sequence", (double)seq);
    if (restart_nonce && restart_nonce[0] != '\0') {
        cJSON_AddStringToObject(root, "restart_nonce", restart_nonce);
    }

    cJSON *metrics = cJSON_AddArrayToObject(root, "metrics");

    cJSON *m_cpu = cJSON_CreateObject();
    cJSON_AddStringToObject(m_cpu, "name", "system.cpu.utilization");
    cJSON_AddNumberToObject(m_cpu, "value", m->cpu_utilization);
    cJSON_AddStringToObject(m_cpu, "unit", "percent");
    cJSON_AddItemToArray(metrics, m_cpu);

    double mem_pct = (m->mem_total_bytes > 0) ? (100.0 * (double)m->mem_used_bytes / (double)m->mem_total_bytes) : 0.0;
    cJSON *m_mem = cJSON_CreateObject();
    cJSON_AddStringToObject(m_mem, "name", "system.memory.utilization");
    cJSON_AddNumberToObject(m_mem, "value", mem_pct);
    cJSON_AddStringToObject(m_mem, "unit", "percent");
    cJSON_AddItemToArray(metrics, m_mem);

    double disk_pct = (m->disk_total_bytes > 0) ? (100.0 * (double)m->disk_used_bytes / (double)m->disk_total_bytes) : 0.0;
    cJSON *m_disk = cJSON_CreateObject();
    cJSON_AddStringToObject(m_disk, "name", "system.disk.utilization");
    cJSON_AddNumberToObject(m_disk, "value", disk_pct);
    cJSON_AddStringToObject(m_disk, "unit", "percent");
    cJSON_AddItemToArray(metrics, m_disk);

    cJSON *m_load = cJSON_CreateObject();
    cJSON_AddStringToObject(m_load, "name", "system.load.1m");
    cJSON_AddNumberToObject(m_load, "value", m->load_avg_1m);
    cJSON_AddStringToObject(m_load, "unit", "count");
    cJSON_AddItemToArray(metrics, m_load);

    char *rendered = cJSON_PrintUnformatted(root);
    if (rendered) {
        snprintf(buffer, size, "%s", rendered);
        free(rendered);
    }
    cJSON_Delete(root);
}

int http_request(
    const char *method,
    const char *host,
    int port,
    const char *path,
    const char *auth_token,
    const char *body,
    char *response_body,
    size_t response_size
);

TxResult send_telemetry(const char *host, int port, const char *path, const char *json_payload, const char *auth_token) {
    char response[1024];
    int status = http_request("POST", host, port, path, auth_token, json_payload, response, sizeof(response));
    if (status >= 200 && status < 300) return TX_OK;
    if (status == 401 || status == 403) return TX_AUTH;
    if (status <= 0 || status == 408 || status == 429 || status >= 500) return TX_RETRY;
    return TX_REJECTED;
}

static int write_all_data(
#ifdef SENTRIX_ENABLE_TLS
    int use_tls, SSL *ssl,
#endif
    int sock, const char *data, size_t len
) {
    size_t written = 0;
    while (written < len) {
        int w = 0;
#ifdef SENTRIX_ENABLE_TLS
        if (use_tls && ssl) {
            w = SSL_write(ssl, data + written, (int)(len - written));
        } else {
            w = (int)send(sock, data + written, len - written, 0);
        }
#else
        w = (int)send(sock, data + written, len - written, 0);
#endif
        if (w <= 0) {
            return 0;
        }
        written += (size_t)w;
    }
    return 1;
}

int http_request(
    const char *method,
    const char *host,
    int port,
    const char *path,
    const char *auth_token,
    const char *body,
    char *response_body,
    size_t response_size
) {
    // TLS is enforced by default. Plaintext HTTP transport is strictly blocked unless explicitly allowed by SENTRIX_ALLOW_INSECURE_HTTP=1.
    const char *allow_insecure_env = getenv("SENTRIX_ALLOW_INSECURE_HTTP");
    int allow_insecure = (allow_insecure_env && (strcmp(allow_insecure_env, "1") == 0 || strcasecmp(allow_insecure_env, "true") == 0));

    int use_tls = 1;
    const char *env_tls = getenv("SENTRIX_USE_TLS");
    if (env_tls && (strcmp(env_tls, "0") == 0 || strcasecmp(env_tls, "false") == 0)) {
        use_tls = 0;
    } else if (allow_insecure && port != 443 && port != 8443 && !env_tls) {
        use_tls = 0;
    }

    if (!use_tls && auth_token && auth_token[0] != '\0' && !allow_insecure) {
        fprintf(stderr, "FATAL SECURITY ERROR: Plaintext HTTP transport with credentials blocked! TLS is required by default. To override for local development only, set SENTRIX_ALLOW_INSECURE_HTTP=1.\n");
        return 0;
    }

#ifndef SENTRIX_ENABLE_TLS
    if (use_tls) {
        fprintf(stderr, "FATAL SECURITY ERROR: HTTPS required for port %d, but sentrix-agent was compiled without OpenSSL TLS support.\n", port);
        return 0;
    }
#endif

    struct addrinfo hints, *res;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;

    char port_str[8];
    snprintf(port_str, sizeof(port_str), "%d", port);

    if (getaddrinfo(host, port_str, &hints, &res) != 0) {
        return 0;
    }

    int sock = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
    if (sock < 0) {
        freeaddrinfo(res);
        return 0;
    }

    struct timeval tv;
    tv.tv_sec = 8;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof(tv));
    setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, (const char*)&tv, sizeof(tv));

#ifndef _WIN32
    int flags = fcntl(sock, F_GETFL, 0);
    if (flags >= 0) {
        fcntl(sock, F_SETFL, flags | O_NONBLOCK);
    }

    int conn_rc = connect(sock, res->ai_addr, res->ai_addrlen);
    if (conn_rc < 0) {
        if (errno == EINPROGRESS) {
            fd_set wfds;
            FD_ZERO(&wfds);
            FD_SET(sock, &wfds);
            struct timeval tv_conn = { .tv_sec = 6, .tv_usec = 0 };
            int sel = select(sock + 1, NULL, &wfds, NULL, &tv_conn);
            if (sel <= 0) {
                close(sock);
                freeaddrinfo(res);
                return 0;
            }
            int so_error = 0;
            socklen_t len = sizeof(so_error);
            getsockopt(sock, SOL_SOCKET, SO_ERROR, &so_error, &len);
            if (so_error != 0) {
                close(sock);
                freeaddrinfo(res);
                return 0;
            }
        } else {
            close(sock);
            freeaddrinfo(res);
            return 0;
        }
    }
    if (flags >= 0) {
        fcntl(sock, F_SETFL, flags);
    }
#else
    if (connect(sock, res->ai_addr, res->ai_addrlen) < 0) {
        close(sock);
        freeaddrinfo(res);
        return 0;
    }
#endif

    freeaddrinfo(res);

#ifdef SENTRIX_ENABLE_TLS
    SSL *ssl = NULL;
    if (use_tls) {
        SSL_CTX *ctx = get_ssl_context();
        if (!ctx) {
            close(sock);
            return 0;
        }

        ssl = SSL_new(ctx);
        if (!ssl) {
            close(sock);
            return 0;
        }

        SSL_set_fd(ssl, sock);
        // Set Server Name Indication (SNI)
        SSL_set_tlsext_host_name(ssl, host);

        // Enforce TLS certificate hostname verification
        if (SSL_set1_host(ssl, host) != 1) {
            fprintf(stderr, "TLS Error: Failed to configure hostname verification for %s\n", host);
            SSL_free(ssl);
            close(sock);
            return 0;
        }

        if (SSL_connect(ssl) <= 0) {
            unsigned long err = ERR_get_error();
            char err_buf[256];
            ERR_error_string_n(err, err_buf, sizeof(err_buf));
            fprintf(stderr, "TLS Handshake failed to %s:%d: %s\n", host, port, err_buf);
            SSL_free(ssl);
            close(sock);
            return 0;
        }

        // Verify certificate if peer validation is required
        if (SSL_get_verify_mode(ssl) != SSL_VERIFY_NONE) {
            long verify_res = SSL_get_verify_result(ssl);
            if (verify_res != X509_V_OK) {
                fprintf(stderr, "TLS Verification Failed: code %ld (%s)\n",
                        verify_res, X509_verify_cert_error_string(verify_res));
                SSL_shutdown(ssl);
                SSL_free(ssl);
                close(sock);
                return 0;
            }
        }
    }
#endif

    char headers[2048];
    int body_len = body ? (int)strlen(body) : 0;

    if (auth_token && auth_token[0] != '\0') {
        snprintf(
            headers,
            sizeof(headers),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Authorization: Bearer %s\r\n"
            "Content-Type: application/json\r\n"
            "Content-Length: %d\r\n"
            "Connection: close\r\n\r\n",
            method,
            path,
            host,
            auth_token,
            body_len
        );
    } else {
        snprintf(
            headers,
            sizeof(headers),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Content-Type: application/json\r\n"
            "Content-Length: %d\r\n"
            "Connection: close\r\n\r\n",
            method,
            path,
            host,
            body_len
        );
    }

    // Write HTTP headers
    if (!write_all_data(
#ifdef SENTRIX_ENABLE_TLS
        use_tls, ssl,
#endif
        sock, headers, strlen(headers)
    )) {
#ifdef SENTRIX_ENABLE_TLS
        if (use_tls && ssl) {
            SSL_shutdown(ssl);
            SSL_free(ssl);
        }
#endif
        close(sock);
        return 0;
    }

    // Write HTTP body without fixed-buffer truncation
    if (body && body_len > 0) {
        if (!write_all_data(
#ifdef SENTRIX_ENABLE_TLS
            use_tls, ssl,
#endif
            sock, body, (size_t)body_len
        )) {
#ifdef SENTRIX_ENABLE_TLS
            if (use_tls && ssl) {
                SSL_shutdown(ssl);
                SSL_free(ssl);
            }
#endif
            close(sock);
            return 0;
        }
    }

    char buffer[8192];
    size_t stored = 0;
    ssize_t n = 0;

    if (response_body && response_size > 0) {
        response_body[0] = '\0';
    }

    while (1) {
#ifdef SENTRIX_ENABLE_TLS
        if (use_tls && ssl) {
            n = SSL_read(ssl, buffer, sizeof(buffer));
        } else {
            n = recv(sock, buffer, sizeof(buffer), 0);
        }
#else
        n = recv(sock, buffer, sizeof(buffer), 0);
#endif

        if (n <= 0) {
            break;
        }

        if (response_body && response_size > 1 && stored < response_size - 1) {
            size_t avail = (response_size - 1) - stored;
            size_t to_copy = (size_t)n < avail ? (size_t)n : avail;
            memcpy(response_body + stored, buffer, to_copy);
            stored += to_copy;
        }

        if (!response_body || response_size <= 1 || stored >= response_size - 1) {
            break;
        }
    }

#ifdef SENTRIX_ENABLE_TLS
    if (use_tls && ssl) {
        SSL_shutdown(ssl);
        SSL_free(ssl);
    }
#endif

    close(sock);

    if (!response_body || response_size == 0) {
        return 0;
    }

    response_body[stored] = '\0';

    int status = 0;
    if (sscanf(response_body, "HTTP/1.%*d %d", &status) != 1) {
        status = 0;
    }

    char *header_end = strstr(response_body, "\r\n\r\n");
    if (header_end) {
        size_t header_len = (size_t)(header_end - response_body) + 4;
        size_t body_received = stored > header_len ? stored - header_len : 0;

        if (body_received > 0) {
            memmove(response_body, response_body + header_len, body_received);
        }

        response_body[body_received] = '\0';
    }

    return status;
}
