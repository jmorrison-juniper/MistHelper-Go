# Actual interface screens

## Scope

The terminal menu is the app's user interface. The HTTP service is **not** a
graphical dashboard: `/` returns service status JSON and `/health` returns
health JSON. Graphical web dashboard screenshots: **N/A, no dashboard exists
in this build**. Do not substitute mockups for screenshots.

These PNG files are screenshots of the real executable running in a PTY
displayed by ttyd's browser terminal. ttyd is a capture tool, not part of the
product. No text was drawn into a mock interface.

## Menu

![Actual MistHelper-Go terminal menu](screenshots/menu.png)

The menu groups entries by category. This view shows part of the real menu;
the terminal scrollback contains the other categories. Registered entries
are not proof that their handlers are implemented.

## Unknown option

![Actual response to nonnumeric menu input](screenshots/unknown-option.png)

Entering `not-a-number` prints guidance and returns to the menu.

## Placeholder operation

![Actual placeholder response for operation 1](screenshots/stub-operation.png)

Entering `1` prints that List Organisation Sites is not yet implemented,
then returns to the menu. This is not an API result.

## Capture method

Captured on 2026-10-05 UTC from source commit
`ea15a7942964c0457ddeb9c325df22b14d49614e`, using Go 1.27.1.
The source code was unchanged by this documentation update.

1. Build `cmd/misthelper` with `CGO_ENABLED=0` in the Go builder container.
2. Run the executable in a temporary Alpine container through ttyd 1.7.7.
   Bind the capture listener to host loopback only.
3. Use a dummy API token, all-zero organization ID, and a disposable SSH
   password. Attach the capture container to an internal-only Podman network,
   with no default route or external DNS resolution.
4. Open the local terminal with Chromium through Playwright. Capture the
   menu, type `not-a-number`, then type `1`. Use terminal scrollback to show
   each response together with the adjacent real menu.
5. Save the screenshots under `docs/screenshots/`. Stop and remove the
   temporary capture container and its network.

Do not select option 26: it calls the Mist API. No live Mist API was used for
these captures. No customer data, credentials, or successful export results
are shown. The captured unknown-option and stub responses can be compared
with the existing menu and entrypoint tests.
