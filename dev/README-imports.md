# Imports Staging

`dev/imports/` is a local staging area for imported artifacts and intake files.

Rules:
- Keep raw imports out of git.
- Use source-specific subfolders to avoid clutter.
- If an imported file becomes reusable product/test data, promote it into a tracked location with clear ownership and purpose.
- Do not rely on files under `dev/imports/` as part of a committed contract.
