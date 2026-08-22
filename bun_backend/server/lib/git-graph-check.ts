import { parseMergeSubject } from "./git-graph"

const cases: [string, string, string][] = [
  ["Merge branch 'feature/login' into develop", "feature/login", "develop"],
  ["Merge pull request #12 from acme/feature/x", "feature/x", ""],
  ["Merged in feature/x (pull request #9)", "feature/x", ""],
  ["regular commit", "", ""],
]
for (const [input, src, dst] of cases) {
  const got = parseMergeSubject(input)
  if (got[0] !== src || got[1] !== dst) {
    throw new Error(`${input}: got ${got} want ${src}/${dst}`)
  }
}
console.log("git-graph parse ok")
