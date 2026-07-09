"""Shared CLI help text for GoLPy."""

from __future__ import annotations

import argparse

TUI_SETUP_CONTROLS_MARKUP = """[bold]Setup screen[/]
  [cyan]W[/] / [cyan]I[/]     Wrapped / Infinite mode
  [cyan]↑[/] / [cyan]↓[/]     Select pattern
  [cyan]←[/] / [cyan]→[/]     Speed (10–200)
  [cyan]D[/]         Toggle density (Low / High)
  [cyan]Enter[/] / [cyan]S[/]  Start simulation
  [cyan]C[/]         Simulation controls (this help)
  [cyan]Q[/]         Quit"""

TUI_PLAY_CONTROLS_MARKUP = """[bold]Simulation[/]
  [cyan]Space[/]     Play / Pause
  [cyan]N[/]         Step one generation
  [cyan]R[/]         Reset (reload pattern)
  [cyan]Q[/]         Back to setup
  [cyan]+[/] / [cyan]-[/]     Speed (10–200)
  [cyan]C[/]         Show controls (this help)
  [cyan]H[/]         Toggle HUD (Pop/Step counters)
  [cyan]E[/]         Toggle edit/selection mode (pauses)
  [cyan]T[/]         Toggle cell under cursor (edit mode)
  [cyan],[/]         Save layout (in-memory snapshot)
  [cyan].[/]         Restore saved layout

[bold]Edit mode[/]
  [cyan]↑↓←→[/] or [cyan]WASD[/]  Infinite: pan field under fixed center cursor
  [cyan]↑↓←→[/] or [cyan]WASD[/]  Wrapped: move cursor on toroidal grid
  [dim]High density: cursor highlights one half of ▄; stats bar shows @ x,y[/]

[bold]Infinite mode[/]
  [cyan]F[/]         Toggle auto-follow population (on/off)
  [cyan]↑↓←→[/] or [cyan]WASD[/]  Pan viewport (paused, or while running if follow off)

[dim]Wrapped mode uses the full terminal as a toroidal grid.
Infinite follow pans one cell per 0.5s while centroid is on-screen; snaps if it leaves.[/]"""

TUI_CONTROLS_MODAL_MARKUP = (
    "[bold yellow]GoLPy — Controls[/]\n\n"
    f"{TUI_PLAY_CONTROLS_MARKUP}\n\n"
    "[dim]Esc or Q to close[/]"
)

GUI_HELP_EPILOG = """
pygame GUI (default):
  Space        Play / Pause
  N            Step
  R            Reset
  H            Toggle HUD (Pop/Step counters)
  Click        Toggle cell
  Drag         Pan (infinite always; wrapped when paused)
  +/- / wheel  Zoom (wrapped when paused only)
  Toolbar      Pattern picker, mode toggle, M+/MR snapshots, speed

  See README.md for full GUI controls.
"""

TUI_HELP_EPILOG = """
Terminal UI (-tui or gol-tui.exe):
  gol.py -tui [--mode wrapped|infinite] [--pattern NAME] [--speed N] [--density low|high]

  Setup screen:
    W / I          Wrapped / Infinite mode
    Up / Down      Select pattern
    Left / Right   Speed (10–200)
    D              Toggle density (Low: 1 char/cell; High: 2 rows/char via ▄)
    Enter / S      Start simulation
    C              Simulation controls overview
    Q              Quit

  Simulation:
    Space          Play / Pause
    N              Step
    R              Reset
    Q              Back to setup
    + / -          Speed
    C              Controls overview
    H              Toggle HUD (Pop/Step counters)
    E              Toggle edit mode (pauses; T toggles cell under cursor)
    ,              Save layout (M+)
    .              Restore saved layout (MR)
    F              Toggle population follow (infinite mode)
    Arrows / WASD  Pan (infinite; paused, or running with follow off)
                   Edit mode: infinite pans field; wrapped moves cursor

  Wrapped: terminal size = toroidal grid (logical rows ×2 in High density).
  Infinite: F toggles auto-follow while running (default off).
  High density stacks two logical cells per terminal row using ▄ (bg=upper, fg=lower).
"""

TUI_ONLY_HELP_EPILOG = """
GoLPy terminal edition (gol-tui.exe / gol_tui.py):
  gol_tui [--mode wrapped|infinite] [--pattern NAME] [--speed N] [--density low|high] [-debug]

  Setup screen:
    W / I          Wrapped / Infinite mode
    Up / Down      Select pattern
    Left / Right   Speed (10–200)
    D              Toggle density (Low: 1 char/cell; High: 2 rows/char via ▄)
    Enter / S      Start simulation
    C              Simulation controls overview
    Q              Quit

  Simulation:
    Space          Play / Pause
    N              Step
    R              Reset
    Q              Back to setup
    + / -          Speed
    C              Controls overview
    H              Toggle HUD (Pop/Step counters)
    E              Toggle edit mode (pauses; T toggles cell under cursor)
    ,              Save layout (M+)
    .              Restore saved layout (MR)
    F              Toggle population follow (infinite mode)
    Arrows / WASD  Pan (infinite; paused, or running with follow off)
                   Edit mode: infinite pans field; wrapped moves cursor

  Wrapped: terminal size = toroidal grid (logical rows ×2 in High density).
  Infinite: F toggles auto-follow while running (default off).
  High density stacks two logical cells per terminal row using ▄ (bg=upper, fg=lower).

  For the pygame window edition, use gol.py or gol.exe instead.
"""


def build_parser(*, tui_only: bool = False) -> argparse.ArgumentParser:
    if tui_only:
        description = "GoLPy — Conway's Game of Life (terminal)"
        epilog = TUI_ONLY_HELP_EPILOG
    else:
        description = "GoLPy — Conway's Game of Life"
        epilog = GUI_HELP_EPILOG + TUI_HELP_EPILOG

    parser = argparse.ArgumentParser(
        description=description,
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=epilog,
    )
    parser.add_argument(
        "--mode",
        choices=("wrapped", "infinite"),
        default="wrapped",
        help="grid mode (default: wrapped)",
    )
    parser.add_argument(
        "--pattern",
        metavar="NAME",
        help="pre-select pattern on setup screen (TUI) or load on startup (GUI)",
    )
    parser.add_argument(
        "--speed",
        type=int,
        default=100,
        metavar="N",
        help="simulation speed 10–200 (default: 100)",
    )
    parser.add_argument(
        "--density",
        choices=("low", "high"),
        default="low",
        help="TUI cell density: low (1 char/cell) or high (2 rows/char via half-blocks)",
    )
    parser.add_argument(
        "-debug",
        action="store_true",
        help="log step stats every 100 generations to stderr",
    )
    if not tui_only:
        parser.add_argument(
            "-tui",
            action="store_true",
            help="terminal UI (Textual); no pygame window",
        )
    return parser
