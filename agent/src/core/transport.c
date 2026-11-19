#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <netdb.h>
#include <time.h>
#include "collector.h"
#include "cJSON.h"

void build_telemetry_json(char *buffer, size_t size, const char *agent_id, uint64_t seq, const SystemMetrics *m) {
    if (!buffer || size == 0) return;

    time_t now = time(NULL);
    struct tm *tm_info = gmtime(&now);
    char timestamp[32];
    strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%SZ", tm_info);

    cJSON *root = cJSON_CreateObject();
    cJSON_AddStringToObject(root, "agent_id", agent_id ? agent_id : "");
    cJSON_AddStringToObject(root, "timestamp", timestamp);
    cJSON_AddNumberToObject(root, "sequence", (double)seq);

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

int send_telemetry(const char *host, int port, const char *path, const char *json_payload, const char *auth_token) {
    char response[1024];
    int status = http_request("POST", host, port, path, auth_token, json_payload, response, sizeof(response));
    return (status >= 200 && status < 300) ? 0 : -1;
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

    if (connect(sock, res->ai_addr, res->ai_addrlen) < 0) {
        close(sock);
        freeaddrinfo(res);
        return 0;
    }

    freeaddrinfo(res);

    char request[8192];
    int body_len = body ? (int)strlen(body) : 0;

    if (auth_token && auth_token[0] != '\0') {
        snprintf(
            request,
            sizeof(request),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Authorization: Bearer %s\r\n"
            "Content-Type: application/json\r\n"
            "Content-Length: %d\r\n"
            "Connection: close\r\n\r\n"
            "%s",
            method,
            path,
            host,
            auth_token,
            body_len,
            body ? body : ""
        );
    } else {
        snprintf(
            request,
            sizeof(request),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Content-Type: application/json\r\n"
            "Content-Length: %d\r\n"
            "Connection: close\r\n\r\n"
            "%s",
            method,
            path,
            host,
            body_len,
            body ? body : ""
        );
    }

    send(sock, request, strlen(request), 0);

    char buffer[8192];
    size_t total = 0;
    ssize_t n;

    while ((n = recv(sock, buffer, sizeof(buffer), 0)) > 0) {
        if (total + n < response_size) {
            memcpy(response_body + total, buffer, n);
        }

        total += n;

        if (total >= response_size) {
            break;
        }
    }

    close(sock);

    if (response_size > 0) {
        response_body[total < response_size ? total : response_size - 1] = '\0';
    }

    int status = 0;
    if (sscanf(response_body, "HTTP/1.%*d %d", &status) != 1) {
        status = 0;
    }

    char *header_end = strstr(response_body, "\r\n\r\n");

    if (header_end) {
        size_t header_len = (size_t)(header_end - response_body) + 4;
        size_t body_received = total > header_len ? total - header_len : 0;

        if (body_received > 0) {
            memmove(response_body, response_body + header_len, body_received);
        }

        if (body_received < response_size) {
            response_body[body_received] = '\0';
        }
    }

    return status;
}
