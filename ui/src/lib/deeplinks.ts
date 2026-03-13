/**
 * Detects whether a set of file paths uses Windows format.
 * Returns true if any path contains a backslash or starts with a drive letter.
 */
export function detectWindowsPaths(paths: string[]): boolean {
  return paths.some((p) => /\\/.test(p) || /^[A-Za-z]:/.test(p))
}

/**
 * Builds a GitHub blob URL.
 * @param repoName - The GitHub repo slug in "org/repo" format.
 * @param filePath - Relative file path within the repository.
 * @param line - Optional line number appended as `#Ln`.
 */
export function githubUrl(repoName: string, filePath: string, line?: number): string {
  const base = `https://github.com/${repoName}/blob/main/${filePath}`
  return line !== undefined ? `${base}#L${line}` : base
}

/**
 * Builds a VS Code deep-link URL.
 *
 * Pass the full absolute file path. The prefix `vscode://file/` is prepended
 * so that:
 * - Unix paths (starting with `/`) produce a double-slash:
 *   `vscode://file//abs/path:line`
 * - Windows paths (starting with a drive letter) produce:
 *   `vscode://file/C:/abs/path:line`
 *
 * @param absolutePath - Absolute path to the file (Unix or Windows format).
 * @param line - Optional line number appended as `:n`.
 */
export function vscodeUrl(absolutePath: string, line?: number): string {
  const lineStr = line !== undefined ? `:${line}` : ''
  return `vscode://file/${absolutePath}${lineStr}`
}

/**
 * Copies text to the system clipboard.
 * Callers should handle the rejection if the Clipboard API is unavailable.
 */
export function copyToClipboard(text: string): Promise<void> {
  return navigator.clipboard.writeText(text)
}
