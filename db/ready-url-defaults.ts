/**
 * Built-in ready-URL patterns seeded into the DB on startup.
 * Capture groups (named): `url` (full URL) and/or `port` (→ http://localhost:{port}).
 *
 * `key` is a stable id used to seed / upgrade defaults without duplicating rows.
 */

export type ReadyUrlPatternSeed = {
  key: string
  label: string
  pattern: string
  flags: string
}

export const DEFAULT_READY_URL_PATTERNS: ReadyUrlPatternSeed[] = [
  {
    key: "next-local",
    label: "Next.js Local",
    pattern: String.raw`-\s*Local:\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "next-ready",
    label: "Next.js Ready",
    pattern: String.raw`\bReady (?:in .+ )?on\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "dev-local",
    label: "Dev server Local",
    pattern: String.raw`\bLocal:\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "dev-network",
    label: "Dev server Network",
    pattern: String.raw`\bNetwork:\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "spring-tomcat",
    label: "Spring Boot Tomcat",
    pattern: String.raw`Tomcat started on port(?:\(s\))?:\s*(?<port>\d+)`,
    flags: "i",
  },
  {
    key: "spring-netty",
    label: "Spring Boot Netty",
    pattern: String.raw`Netty started on port\s+(?<port>\d+)`,
    flags: "i",
  },
  {
    key: "spring-tomcat-init",
    label: "Spring Boot Tomcat init",
    pattern: String.raw`Tomcat initialized with port(?:\(s\))?:\s*(?<port>\d+)`,
    flags: "i",
  },
  {
    key: "dotnet-listening",
    label: ".NET Kestrel",
    pattern: String.raw`Now listening on:\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "django-dev",
    label: "Django",
    pattern: String.raw`Starting development server at\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "listening-on-url",
    label: "Listening on URL",
    pattern: String.raw`\bListening on\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
  {
    key: "listening-on-port",
    label: "Listening on port",
    pattern: String.raw`\blisten(?:ing)? on port\s+(?<port>\d+)`,
    flags: "i",
  },
  {
    key: "serving-http-port",
    label: "Serving HTTP",
    pattern: String.raw`\bserving HTTP on\s+(?:\S*?:)?(?<port>\d+)`,
    flags: "i",
  },
  {
    key: "generic-url",
    label: "Generic URL",
    pattern: String.raw`\b(?:running|started|available|serving)\s+(?:at|on)\s+(?<url>https?:\/\/\S+)`,
    flags: "i",
  },
]
