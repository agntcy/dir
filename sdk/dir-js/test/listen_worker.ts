import { workerData } from 'worker_threads';
import { spawnSync } from 'node:child_process';
import { env } from 'node:process';

async function pullRecords() {
    const shell_env = env;

    let commandArgs = ["pull", workerData.recordRef.cid];

    if (workerData.spiffeEndpointSocket !== '') {
        commandArgs.push(...["--spiffe-socket-path", workerData.spiffeEndpointSocket]);
    }

    for (let count = 0; count < 10; count++) {
        // Execute command
        spawnSync(
            `${workerData.dirctlPath}`, commandArgs,
            { env: { ...shell_env }, encoding: 'utf8', stdio: 'pipe' },
        );

        await new Promise(resolve => setTimeout(resolve, 1000));
    }
}

pullRecords()
