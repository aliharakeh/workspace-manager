/**
 * Strip CSI / OSC ANSI escape sequences from CLI output.
 * Also removes orphaned CSI tails (e.g. "[33m") left when ESC was lost at a chunk boundary.
 */
const ANSI_RE =
  // eslint-disable-next-line no-control-regex -- intentional: match ESC / CSI sequences
  /(?:\x1B[@-Z\\-_]|\x1B\[[0-?]*[ -/]*[@-~]|\x1B\][^\x07\x1B]*(?:\x07|\x1B\\)|\[[\d;]+[A-Za-z])/g

export function stripAnsi(text: string): string {
  return text.replace(ANSI_RE, "")
}
