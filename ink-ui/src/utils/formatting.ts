/**
 * Truncates a string to a maximum number of lines.
 * If the string exceeds the limit, it returns the first maxLines lines
 * followed by a message indicating how many lines were hidden.
 * 
 * @param content The string content to truncate
 * @param maxLines The maximum number of lines to show
 * @returns The truncated string
 */
export function truncateOutput(content: string, maxLines: number): string {
    const lines = content.split('\n');
    if (lines.length <= maxLines) {
        return content;
    }
    return lines.slice(0, maxLines).join('\n') + `\n... (${lines.length - maxLines} lines hidden)`;
}
