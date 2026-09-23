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
| `static/bidder-info/insticator.yaml` | Carries the **staging** hosts. The adapter branch and upstream both carry production hosts, so rebuilding `stage` from the adapter branch silently points staging at production. See [Staging endpoints](#staging-endpoints). |

Everything else — the adapter, its tests, its YAML, and the rest of the tree — is
fair game and is expected to change.

## Updating the adapter

The adapter work is developed against upstream and opened as a pull request with
base `prebid/master`. To get it deployed:

1. Bring `prebid/master` up to date with `upstream/master`.
2. Put the adapter commits on top of that.
3. Rebuild `stage` as: that content, plus the files in the table above. The
   adapter branch's `static/bidder-info/insticator.yaml` carries production
   hosts, so re-apply the staging values from [Staging endpoints](#staging-endpoints).
4. Push `stage`.

`stage` therefore differs from the adapter branch by exactly the files listed
above. Anything else appearing in a diff between them means the two have drifted
and should be reconciled before shipping.

## Staging endpoints

`stage` must point at staging, never production. Nothing in the deployment
overrides these — no environment variable, and no `adapters.insticator.*` block in
the mounted `pbs.yaml` — so whatever is baked into
`static/bidder-info/insticator.yaml` is what the running server calls.

| Field                     | Staging (`stage`)                            | Production (`master`)                         |
|---------------------------|----------------------------------------------|-----------------------------------------------|
| `endpoint`                | `https://ex-v2.hunchme.com/v1/prebidserver`  | `https://ex.ingage.tech/v1/prebidserver`      |
| `extra_info.app_endpoint` | `https://ex-v2.hunchme.com/v1/prebidserver`  | `https://aex.ingage.tech/v1/prebidserver`     |
| `userSync.iframe.url`     | `https://sync.hunchme.com`                   | `https://usync.ingage.tech`                   |

Staging runs one exchange (`ex-v2.hunchme.com`), so the app endpoint is the same
host as the web one; production splits them across `ex` and `aex`. Use
`ex-v2.hunchme.com`, not `ex.hunchme.com` — the latter is the retired v1 exchange
and is not running.

If these are left pointing at production, staging still builds, deploys and
reports healthy: `/status` returns 204 and the adapter appears in
`/info/bidders`. Only the bids disappear, because production has no inventory for
the staging test ad units and answers every request with 204. Check the endpoint
in the adapter's `httpcalls` before assuming a no-bid is a bidding problem.

## Keeping the adapter upstreamable

The adapter is intended to be merged upstream, so the fork should not accumulate
changes to shared code. If a change outside `adapters/insticator/` becomes
necessary, record it here with the reason — otherwise the next person to sync the
fork will drop it without knowing it mattered.
