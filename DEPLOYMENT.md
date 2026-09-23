# Deployment

This fork exists to build and ship the Insticator adapter. Everything outside the
files listed below is a straight copy of upstream `prebid/prebid-server` and can
be replaced wholesale when the fork is brought up to date.

## How a deploy happens

CircleCI watches two branches and builds a Docker image from the repository root
on each push:

| Branch   | Image repository                       |
|----------|----------------------------------------|
| `stage`  | `insticator-prebid-server-stage`       |
| `master` | `insticator-prebid-server-master`      |

Pushing to `stage` is what triggers a staging deploy. There is no other trigger,
and nothing deploys from a pull request.

## Files that must not be removed

| Path                    | Why |
|-------------------------|-----|
| `.circleci/config.yml`  | The only deployment file in the fork. Without it nothing builds or ships. It is not present upstream, so a hard reset onto upstream will delete it. |
| `Dockerfile`            | Referenced by the CircleCI job as `dockerfile: Dockerfile` with `path: .`. It is currently identical to upstream's — keep it that way unless the build genuinely needs to differ, and if it does, say so here. |
| `DEPLOYMENT.md`         | This file. |

Everything else — the adapter, its tests, its YAML, and the rest of the tree — is
fair game and is expected to change.

## Updating the adapter

The adapter work is developed against upstream and opened as a pull request with
base `prebid/master`. To get it deployed:

1. Bring `prebid/master` up to date with `upstream/master`.
2. Put the adapter commits on top of that.
3. Rebuild `stage` as: that content, plus the files in the table above.
4. Push `stage`.

`stage` therefore differs from the adapter branch by exactly the files listed
above. Anything else appearing in a diff between them means the two have drifted
and should be reconciled before shipping.

## Keeping the adapter upstreamable

The adapter is intended to be merged upstream, so the fork should not accumulate
changes to shared code. If a change outside `adapters/insticator/` becomes
necessary, record it here with the reason — otherwise the next person to sync the
fork will drop it without knowing it mattered.
