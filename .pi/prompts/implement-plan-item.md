---
description: Implement one item from the development plan in small commits and push its branch
argument-hint: "[section and item, title, or selection guidance]"
---

Implement one top-level item from `notes/plan.md` on a new branch and push that branch to GitHub.

Selection guidance: ${ARGUMENTS:-Choose the next suitable unfinished item yourself.}

Carry out the complete workflow below. Do not merely describe what should be done.

## 1. Perform preflight checks

- Read `AGENTS.md`, `notes/plan.md`, and any relevant repository documentation.
- Inspect the current branch, worktree status, recent history, remotes, and the repository's default branch.
- Verify that `origin` is an SSH GitHub remote, then fetch from it to confirm access and refresh the remote-tracking refs. Use only Git over SSH for remote operations; do not use `gh` or the GitHub API. Do not pull, merge, reset, rebase, or otherwise alter local branches during preflight.
- Require a clean worktree before creating the task branch. If tracked or untracked work is present, stop and explain what must be resolved; do not stash, discard, or include it.
- Require the local default branch to match its upstream exactly, with no unpushed commits and no commits missing locally. Stop and explain any divergence rather than resolving it automatically.

## 2. Configure the commit author name

- Read the current model ID from the `PI_MODEL` environment variable. Stop if it is empty.
- Define the expected name and set the dev container's global Git user name:

  ```bash
  expected_name="${PI_MODEL} (acting on behalf of Sunny Fawcett)"
  git config --global user.name "$expected_name"
  test "$(git config --global --get user.name)" = "$expected_name"
  test "$(git config --get user.name)" = "$expected_name"
  git config --get user.email >/dev/null
  ```

- Check the origin of the effective `user.name`. If repository-local or other higher-priority configuration overrides the expected name, stop and explain the conflict; do not modify that overriding configuration.
- If there is no effective `user.email`, stop and explain the problem. Do not change `user.email` or any other global identity configuration.

## 3. Select one plan item

- Apply the selection guidance and choose one unfinished top-level item that is reasonably scoped for one branch of work and which satisfies the selection guidance.
- If no items satisfy the selection guidance, stop and alert the user.
- Do not select work already implemented on the default branch.
- Prefer prerequisite refactoring before behavior changes that depend on it.
- If the item requires an unresolved product, policy, or design decision, stop and ask the user rather than choosing a policy implicitly.
- State the selected item and a short reason for choosing it, then continue without waiting for confirmation.

## 4. Create the task branch

- After preflight succeeds, switch to the local default branch.
- From the default branch, create and switch to a branch named `<type>/<short-kebab-case-topic>`.
- Choose the type by the primary purpose of the complete plan item:
  - `refactor/` for behavior-preserving restructuring;
  - `fix/` for correctness, safety, or usability fixes;
  - `feature/` for new user-facing functionality;
  - `docs/` for documentation-only work;
  - `test/` for test-only work; or
  - `chore/` for tooling and repository maintenance.
- Describe the complete plan item rather than an individual implementation step. Keep the topic concise, normally two to four lowercase words separated by hyphens, and prefer the intended outcome over implementation details.
- Avoid vague topics such as `updates`, `cleanup`, or `work`.
- Confirm that the branch name does not already exist locally or on `origin` before creating it. If it exists, choose a different name; do not overwrite or delete the existing branch.

## 5. Implement the complete item

- Inspect the relevant code and tests before editing.
- Complete every applicable small implementation step listed under the selected item. If blocked, stop and ask the user rather than pushing an incomplete branch. For conditional steps such as “consider,” record why they were unnecessary when they do not apply.
- Preserve existing behavior unless the plan item explicitly calls for a behavior change.
- Follow `AGENTS.md` and the repository's established style.
- Keep implementation changes limited to the selected item. Do not perform unrelated cleanup.
- Do not edit `notes/plan.md` merely to record completion unless the user explicitly requests it.
- Add or adjust tests when they provide useful coverage under the repository's testing guidelines.

## 6. Make small commits

- Commit each coherent implementation step separately when the repository remains valid and understandable at that boundary.
- Prefer one commit per plan step, but combine tightly coupled steps when splitting them would produce a broken, misleading, or mechanical-only commit.
- Keep tests with the behavior or refactoring they verify.
- Before each commit:
  - inspect the staged diff;
  - ensure it contains no unrelated changes;
  - recompute the expected name from the current `PI_MODEL`, update the global name if the model changed, and verify that the effective `user.name` equals the expected name; and
  - format changed files and run the checks appropriate to that step.
- Use concise imperative commit subjects that describe the change.
- Do not rewrite commit history, including with `git commit --amend` or interactive rebase, unless the user explicitly requests it.

## 7. Record follow-up findings

- At any point in the workflow, stop and notify the user immediately if a discovery indicates a serious security, data-loss, or correctness risk rather than merely adding it to the TODO file.
- Do not expand the selected item's scope to address unrelated discoveries.
- Fix problems introduced by the current work rather than recording them as follow-up items.
- Search `notes/plan.md` and `notes/todo.md` for overlapping entries before recording a finding, reading the relevant sections as needed, and do not add duplicates.
- Add only concrete findings supported by something observed during the work. Do not add unsupported speculation or trivial cleanup.
- Place each finding under `Unplanned items`, then under the relevant package and type: `Refactor`, `Fix`, `Feature`, `Docs`, `Test`, or `Chore`. Use `All packages` for cross-cutting findings, and add a missing package or type heading in the documented order only when needed.
- Keep the observed problem, why it matters, and the plausible solution or next step together in one item.
- Do not reorganize, remove, promote, or otherwise triage existing entries.
- Do not modify `notes/todo.md` when there are no worthwhile findings.
- Commit additions separately with the subject `Document follow-up findings`.

## 8. Review and validate the branch

- Review the complete diff and commit list against the default branch.
- Run `gofmt` on changed Go files and run the full relevant test suite, including `go test ./...` unless there is a clear reason it cannot run.
- If formatting changes files or validation fails, fix the issue in an additional focused commit and repeat this section.
- Check that every applicable step in the selected plan item was completed.
- Confirm that the worktree is clean.

## 9. Push the branch

- Push the branch to `origin` and set its upstream. Never force push.
- Verify that the remote branch exists and points to the same commit as the local branch.
- Do not create a pull request.
- Finish by reporting:
  - the selected development-plan item;
  - the branch name;
  - the commits created;
  - the tests and checks run;
  - any conditional plan steps judged unnecessary, with the reason;
  - the follow-up findings added to `notes/todo.md`, or that there were none; and
  - the GitHub compare URL the user can open to create a pull request manually, when it can be derived from the `origin` URL and default branch.
