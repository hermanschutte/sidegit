package main

import (
	"encoding/json"
	"os/exec"
)

// PRStatus is the newest GitHub pull request for a branch.
type PRStatus struct {
	Number     int
	State      string // OPEN, MERGED or CLOSED
	Draft      bool
	Checks     string // SUCCESS, FAILURE, ERROR, PENDING or EXPECTED; empty without checks
	Unresolved int    // review threads not resolved yet
}

const prQuery = `query($owner:String!,$name:String!,$branch:String!){
  repository(owner:$owner,name:$name){
    pullRequests(headRefName:$branch,first:1,orderBy:{field:CREATED_AT,direction:DESC}){
      nodes{ number state isDraft
        reviewThreads(first:100){ nodes{ isResolved } }
        commits(last:1){ nodes{ commit{ statusCheckRollup{ state } } } } } } } }`

// FetchPR asks GitHub through the gh CLI for the newest pull request of the branch.
// It returns nil when the branch has no PR, the repo is not on GitHub, or gh fails.
func FetchPR(repoPath, branch string) *PRStatus {
	// gh fills {owner} and {repo} from the git remote of the folder it runs in.
	cmd := exec.Command("gh", "api", "graphql",
		"-F", "owner={owner}", "-F", "name={repo}", "-f", "branch="+branch, "-f", "query="+prQuery)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var resp struct {
		Data struct {
			Repository struct {
				PullRequests struct {
					Nodes []struct {
						Number        int
						State         string
						IsDraft       bool
						ReviewThreads struct {
							Nodes []struct{ IsResolved bool }
						}
						Commits struct {
							Nodes []struct {
								Commit struct {
									StatusCheckRollup *struct{ State string }
								}
							}
						}
					}
				}
			}
		}
	}
	if err := json.Unmarshal(out, &resp); err != nil || len(resp.Data.Repository.PullRequests.Nodes) == 0 {
		return nil
	}

	n := resp.Data.Repository.PullRequests.Nodes[0]
	pr := &PRStatus{Number: n.Number, State: n.State, Draft: n.IsDraft}
	for _, t := range n.ReviewThreads.Nodes {
		if !t.IsResolved {
			pr.Unresolved++
		}
	}
	if len(n.Commits.Nodes) > 0 && n.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
		pr.Checks = n.Commits.Nodes[0].Commit.StatusCheckRollup.State
	}
	return pr
}
