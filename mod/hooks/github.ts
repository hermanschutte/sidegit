import type { EngineInterface } from 'claude-code'

import type { PRStatus } from '../types'

const PR_QUERY = `query($owner:String!,$name:String!,$branch:String!){
  repository(owner:$owner,name:$name){
    pullRequests(headRefName:$branch,first:1,orderBy:{field:CREATED_AT,direction:DESC}){
      nodes{ number state isDraft
        reviewThreads(first:100){ nodes{ isResolved } }
        commits(last:1){ nodes{ commit{ statusCheckRollup{ state } } } } } } } }`

// fetchPR asks GitHub through the gh CLI for the newest pull request of the branch, as the
// sidegit binary does. It returns null when the branch has no PR, the repo is not on
// GitHub, or gh fails.
export async function fetchPR($: EngineInterface, repoPath: string, branch: string): Promise<PRStatus | null> {
  try {
    // gh fills {owner} and {repo} from the git remote of the folder it runs in
    const run = await $.process.run(
      ['gh', 'api', 'graphql', '-F', 'owner={owner}', '-F', 'name={repo}', '-f', `branch=${branch}`, '-f', `query=${PR_QUERY}`],
      { cwd: repoPath, timeoutMs: 30_000 },
    )
    return run.exitCode === 0 ? parsePR(run.stdout) : null
  } catch {
    return null
  }
}

type Response = {
  data?: {
    repository?: {
      pullRequests?: {
        nodes?: {
          number: number
          state: string
          isDraft: boolean
          reviewThreads?: { nodes?: { isResolved: boolean }[] }
          commits?: { nodes?: { commit?: { statusCheckRollup?: { state: string } | null } }[] }
        }[]
      }
    }
  }
}

export function parsePR(stdout: string): PRStatus | null {
  let resp: Response
  try {
    resp = JSON.parse(stdout) as Response
  } catch {
    return null
  }
  const n = resp.data?.repository?.pullRequests?.nodes?.[0]
  if (!n) return null
  return {
    number: n.number,
    state: n.state,
    draft: n.isDraft,
    checks: n.commits?.nodes?.[0]?.commit?.statusCheckRollup?.state ?? '',
    unresolved: (n.reviewThreads?.nodes ?? []).filter(t => !t.isResolved).length,
  }
}
