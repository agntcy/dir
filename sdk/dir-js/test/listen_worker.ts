import { spawnSync } from 'node:child_process';
import { env } from 'node:process';
import { worker } from 'workerpool';

worker({
    pullRecordsBackground,
});

export async function pullRecordsBackground(cid: string, dirctlPath: string, spiffeEndpointSocket: string) {
    const shell_env = env;

    let commandArgs = ["pull", cid];

    if (spiffeEndpointSocket !== '') {
        commandArgs.push(...["--spiffe-socket-path", spiffeEndpointSocket]);
    }

    for (let count = 0; count < 90; count++) {
        // Execute command
        spawnSync(
            `${dirctlPath}`, commandArgs,
            { env: { ...shell_env }, encoding: 'utf8', stdio: 'pipe' },
        );

        await new Promise(resolve => setTimeout(resolve, 1000));
    }
}
