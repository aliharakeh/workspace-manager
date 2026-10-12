/** Decodes the base64 terminal bytes the Go side sends. */
export function decodeTerminalData(data: string): Uint8Array {
  if (!data) return new Uint8Array(0)
  const binary = atob(data)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes
}
