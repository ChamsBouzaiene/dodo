import { renderHook, act } from '@testing-library/react';
import { useConversation } from '../hooks/useConversation.js';
import fs from 'fs';
import path from 'path';

/**
 * Performance Benchmark Runner
 * Measures O(N) scaling and memory overhead
 */
async function runBenchmarks() {
    console.log("🚀 Starting Performance Benchmarks...");
    const results: any = {
        timestamp: new Date().toISOString(),
        metrics: []
    };

    const iterations = [10, 50, 100, 200, 500];

    for (const count of iterations) {
        console.log(`\n--- Testing with ${count} turns ---`);
        const { result } = renderHook(() => useConversation());

        // 1. Setup Phase
        const setupStart = process.hrtime.bigint();
        act(() => {
            for (let i = 0; i < count; i++) {
                result.current.pushTurn(`User query ${i}`);
                result.current.appendAssistantContent(`Assistant response for turn ${i}. ` + "Token ".repeat(20));
                result.current.markLastTurnDone();
            }
        });
        const setupEnd = process.hrtime.bigint();
        const setupMs = Number(setupEnd - setupStart) / 1000000;

        // 2. Latency Phase (O(N) check)
        // Measure how long it takes to append content to the LAST turn when there are 'count' turns
        const updateStart = process.hrtime.bigint();
        act(() => {
            result.current.pushTurn("Final benchmark query");
            for (let i = 0; i < 100; i++) {
                result.current.appendAssistantContent(`token_${i}`);
            }
        });
        const updateEnd = process.hrtime.bigint();
        const updateMs = Number(updateEnd - updateStart) / 1000000;
        const perTokenLatency = updateMs / 100;

        // 3. Memory Phase
        const heap = process.memoryUsage().heapUsed / 1024 / 1024;

        console.log(`Setup time: ${setupMs.toFixed(2)}ms`);
        console.log(`Avg Token Latency: ${perTokenLatency.toFixed(4)}ms`);
        console.log(`Heap Usage: ${heap.toFixed(2)}MB`);

        results.metrics.push({
            turnCount: count,
            setupTimeMs: setupMs,
            tokenLatencyMs: perTokenLatency,
            heapMb: heap
        });
    }

    const reportPath = path.resolve(process.cwd(), 'performance_results.json');
    fs.writeFileSync(reportPath, JSON.stringify(results, null, 2));
    console.log(`\n✅ Benchmarks complete. Results saved to ${reportPath}`);
}

// In a real environment, this would be run via tsx or similar
// For this task, we can wrap it in a Vitest test if needed, but the logic is here.
runBenchmarks().catch(console.error);
