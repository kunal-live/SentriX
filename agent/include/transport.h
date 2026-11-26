#ifndef SENTRIX_TRANSPORT_H
#define SENTRIX_TRANSPORT_H

/*
 * Outcome of a telemetry POST. The caller's state machine needs to tell these
 * apart: a network blip should be retried, but a frame the server rejected
 * will be rejected forever, and a rejected credential will never recover.
 */
typedef enum {
    TX_OK = 0,        /* 2xx: delivered (includes server-side duplicates) */
    TX_RETRY = -1,    /* network error, timeout, 408, 429 or 5xx: try again later */
    TX_REJECTED = -2, /* other 4xx (e.g. 409 stale sequence, 400, 413): drop the frame */
    TX_AUTH = -3      /* 401/403: credential invalid or revoked; retrying cannot help */
} TxResult;

TxResult send_telemetry(const char *host, int port, const char *path, const char *json_payload, const char *auth_token);

#endif
