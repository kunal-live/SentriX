#include "collector.h"
#include <stdio.h>
#include <string.h>
#include <unistd.h>

int collect_cpu(double *utilization) {
    if (!utilization) return -1;
    *utilization = 0.0;

    FILE *fp = fopen("/proc/stat", "r");
    if (!fp) return -1;

    char line[256] = {0};
    uint64_t user1 = 0, nice1 = 0, system1 = 0, idle1 = 0, iowait1 = 0, irq1 = 0, softirq1 = 0, steal1 = 0;
    uint64_t user2 = 0, nice2 = 0, system2 = 0, idle2 = 0, iowait2 = 0, irq2 = 0, softirq2 = 0, steal2 = 0;

    // First read
    if (!fgets(line, sizeof(line), fp)) {
        fclose(fp);
        return -1;
    }
    if (sscanf(line, "cpu  %lu %lu %lu %lu %lu %lu %lu %lu", 
               &user1, &nice1, &system1, &idle1, &iowait1, &irq1, &softirq1, &steal1) < 4) {
        fclose(fp);
        return -1;
    }
    fclose(fp);

    usleep(250000); // Wait 250ms for delta calculation

    fp = fopen("/proc/stat", "r");
    if (!fp) return -1;
    
    // Second read
    memset(line, 0, sizeof(line));
    if (!fgets(line, sizeof(line), fp)) {
        fclose(fp);
        return -1;
    }
    if (sscanf(line, "cpu  %lu %lu %lu %lu %lu %lu %lu %lu", 
               &user2, &nice2, &system2, &idle2, &iowait2, &irq2, &softirq2, &steal2) < 4) {
        fclose(fp);
        return -1;
    }
    fclose(fp);

    uint64_t total1 = user1 + nice1 + system1 + idle1 + iowait1 + irq1 + softirq1 + steal1;
    uint64_t total2 = user2 + nice2 + system2 + idle2 + iowait2 + irq2 + softirq2 + steal2;
    
    if (total2 <= total1) {
        *utilization = 0.0;
        return 0;
    }

    uint64_t total_delta = total2 - total1;
    uint64_t idle_delta = (idle2 >= idle1) ? (idle2 - idle1) : 0;

    if (total_delta == 0) {
        *utilization = 0.0;
    } else {
        double util = 100.0 * (1.0 - ((double)idle_delta / (double)total_delta));
        if (util < 0.0) util = 0.0;
        if (util > 100.0) util = 100.0;
        *utilization = util;
    }

    return 0;
}
