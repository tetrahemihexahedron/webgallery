---
description: Implement the first todo item, optionally removing a completed item first
argument-hint: "[delete-first]"
---

Implement the first top-level item from `notes/todo.md` on a new branch and push that branch to GitHub. With no argument, implement the current first item. With the exact argument `delete-first`, remove the completed first item and then implement the new first item.

Requested route: `${ARGUMENTS:-implement-first}`

Carry out the complete workflow below. Do not merely describe what should be done.

## 1. Perform preflight checks

- Validate the requested route before making any repository or configuration changes. Accept only `implement-first`, produced when the prompt is invoked without an argument, or `delete-first`, produced by passing that exact argument. If the route is anything else, stop and explain the two valid invocations; do not interpret other text as selection guidance.
- Read `AGENTS.md`, `notes/todo.md`, and any relevant repository documentation.
- Inspect the current branch, worktree status, recent history, remotes, and the repository's default branch.
- Require a clean worktree before performing remote operations or creating the task branch. If tracked or untracked work is present, stop and explain what must be resolved; do not stash, discard, or include it.
- Verify that `origin` is an SSH GitHub remote, then fetch from it to confirm access and refresh the remote-tracking refs. Use only Git over SSH for remote operations; do not use `gh` or the GitHub API. Do not pull, merge, reset, rebase, or otherwise alter local branches during preflight.
- Require the local default branch to exist, track the default branch on `origin`, and match it exactly, with no unpushed commits and no commits missing locally. Stop and explain any divergence rather than resolving it automatically.

## 2. Configure the commit author name

- Read the current model ID from the `PI_MODEL` environment variable. Stop if it is empty.
- Define the expected name and set the dev container's global Git user name:

  ```bash
  expected_name="${PI_MODEL} (acting on behalf of Sunny Fawcett)"
  git config --global user.name "$expected_name"
  test "$(git config --global --get user.name)" = "$expected_name"
  test "$(git config --get user.name)" = "$expected_name"
  test -n "$(git config --get user.email)"
  ```

- Check the origin of the effective `user.name`. If repository-local or other higher-priority configuration overrides the expected name, stop and explain the conflict; do not modify that overriding configuration.
- If there is no effective `user.email`, stop and explain the problem. Do not change `user.email` or any other global identity configuration.

## 3. Apply the requested route

- Switch to the local default branch. Do not merge, cherry-pick, pull, reset, rebase, or otherwise bring implementation work onto it.
- Identify the first top-level item: the first second-level (`##`) section in `notes/todo.md`. If there is no item, stop and alert the user.
- For the `implement-first` route, leave `notes/todo.md` unchanged and continue.
- For the `delete-first` route, treat the argument as the user's authoritative confirmation; do not independently guess whether the item was completed:
  - Remove the first item's entire section from `notes/todo.md` and renumber the remaining numbered top-level headings in order without changing other content or otherwise reorganizing the file.
  - Inspect the diff and ensure it contains only the intended `notes/todo.md` removal and heading renumbering.
  - Run every check required by `notes/todo.md`, currently `go test ./...`, `go vet ./...`, and `staticcheck ./...`. If any check fails, stop without committing.
  - Stage only `notes/todo.md`, inspect the staged diff, and commit it on the local default branch with the subject `Remove completed todo item`.
  - Push the default branch directly to `origin` over SSH without force, verify that the remote default branch points to the same commit as the local branch, and require a clean worktree. If the commit or push fails, stop before selecting or creating another task branch.
  - If removing the item leaves no top-level items, stop and alert the user after pushing the cleanup commit.

## 4. Select the first item

- Select the first top-level item currently in `notes/todo.md`; do not skip or reorder items.
- If it appears to have already been implemented on the default branch, stop and tell the user to rerun the prompt with `delete-first`; do not delete or skip it automatically.
- If the item cannot reasonably fit on one branch, stop and alert the user rather than selecting a later item.
- If the item requires an unresolved product, policy, or design decision, stop and ask the user rather than choosing a policy implicitly.
- State the selected item and continue without waiting for confirmation.

## 5. Create the task branch

- Ensure that the local default branch is checked out and still matches its upstream exactly after the completed-item check.
- Inspect local and remote branch names and relevant history for an existing branch that appears to implement the selected item. If one exists, stop and report it rather than duplicating the work.
- From the default branch, create and switch to a branch named `<type>/<short-kebab-case-topic>`.
- Choose the type by the primary purpose of the complete item:
  - `refactor/` for behavior-preserving restructuring;
  - `fix/` for correctness, safety, or usability fixes;
  - `feature/` for new user-facing functionality;
  - `docs/` for documentation-only work;
  - `test/` for test-only work; or
  - `chore/` for tooling and repository maintenance.
- Describe the complete selected item rather than an individual implementation step. Keep the topic concise, normally two to four lowercase words separated by hyphens, and prefer the intended outcome over implementation details.
- Avoid vague topics such as `updates`, `cleanup`, or `work`.
- Confirm that the branch name does not already exist locally or on `origin` before creating it. If an unrelated branch already uses the proposed name, choose another descriptive name; do not overwrite or delete the existing branch.

## 6. Implement the complete item

- Inspect the relevant code and tests before editing.
- Complete every applicable small implementation step listed under the selected item. If blocked, stop and ask the user rather than pushing an incomplete branch. For conditional steps such as “consider,” record why they were unnecessary when they do not apply.
- Prefer a simpler or better implementation when it still fulfills the selected item. Record any material departure from the described implementation and ask the user before changing the item's scope or intended behavior.
- Preserve existing behavior unless the item explicitly calls for a behavior change.
- Follow `AGENTS.md` and the repository's established style.
- Keep implementation changes limited to the selected item. Do not perform unrelated cleanup.
- Do not edit `notes/todo.md` on the task branch merely to record completion.
- Add or adjust tests when they provide useful coverage under the repository's testing guidelines in `AGENTS.md`.

## 7. Make small commits

- Make one or more independently passing commits for every numbered implementation step in the selected item. Do not combine separate numbered steps into one commit.
- Split a step further only at coherent boundaries where the repository remains valid and understandable. When a migration needs multiple commits, make additive changes that keep intermediate commits working, and remove any temporary compatibility adapters before finishing the branch.
- Work on one commit boundary at a time so later uncommitted work does not hide whether the next commit passes independently.
- Keep tests with the behavior or refactoring they verify.
- Before each task-branch commit:
  - format changed Go files, if any;
  - run every check required by `notes/todo.md`, currently `go test ./...`, `go vet ./...`, and `staticcheck ./...`;
  - stage only the coherent change intended for that commit; and
  - inspect the staged diff and worktree status to ensure the commit is complete and contains no unrelated changes.
- Use concise imperative commit subjects that describe the change.
- Do not rewrite commit history, including with `git commit --amend` or interactive rebase, unless the user explicitly requests it.

## 8. Record follow-up findings

- At any point in the workflow, stop and notify the user immediately if a discovery indicates a serious security, data-loss, or correctness risk rather than merely adding it to `notes/backlog.md`.
- Do not expand the selected item's scope to address unrelated discoveries.
- Fix problems introduced by the current work rather than recording them as follow-up items.
- Search `notes/backlog.md` and `notes/todo.md` for overlapping entries before recording a finding, reading the relevant sections as needed, and do not add duplicates.
- Add only concrete findings supported by something observed during the work. Do not add unsupported speculation or trivial cleanup.
- Place each finding under `Postponed`, then under the relevant package and type: `Refactor`, `Fix`, `Feature`, `Docs`, `Test`, or `Chore`. Use `All packages` for cross-cutting findings, and add a missing package or type heading in the documented order only when needed.
- Keep the observed problem, why it matters, and the plausible solution or next step together in one item.
- Do not reorganize, remove, promote, or otherwise triage existing entries.
- Do not modify `notes/backlog.md` when there are no worthwhile findings.
- Commit additions separately with the subject `Document follow-up findings`.

## 9. Review and validate the branch

- Review the complete diff and commit list against the default branch.
- Run `gofmt` on all changed Go files, then run `go test ./...`, `go vet ./...`, and `staticcheck ./...`.
- If formatting changes files or validation fails, fix the issue in an additional focused commit following the commit workflow above, then repeat this section.
- Check that every applicable step in the selected item was completed and represented by at least one independently passing commit.
- Confirm that the worktree is clean.

## 10. Push the branch

- Push the branch to `origin` and set its upstream. Never force push.
- Verify that the remote branch exists and points to the same commit as the local branch.
- Do not create a pull request.
- Finish by reporting:
  - whether the first item was removed, and the cleanup commit if one was created;
  - the selected item;
  - the branch name;
  - the commits created;
  - the tests and checks run;
  - any conditional steps judged unnecessary, with the reason;
  - any material adjustments made to the described implementation, or that there were none;
  - the follow-up findings added to `notes/backlog.md`, or that there were none; and
  - the GitHub compare URL the user can open to create a pull request manually, when it can be derived from the `origin` URL and default branch.
