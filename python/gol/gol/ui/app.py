"""Pygame front-end for GoLPy."""

from __future__ import annotations

import sys
import time

import pygame

from gol.colors import cell_rgb
from gol.engine import GameOfLife, Mode, Snapshot
from gol.icon import set_window_icon
from gol.patterns import WRAPPED_MIN_AXIS, pattern_names, wrapped_grid_layout
from gol.ui import controls

BASE_DELAY_MS = 16
BORDER_HUE_INCREMENT = 1.5
FRAME_BORDER = 2
MIN_WINDOW = (480, 400)


class GolApp:
    def __init__(
        self,
        *,
        mode: Mode = "wrapped",
        pattern: str | None = None,
        speed: int = 100,
        debug: bool = False,
    ) -> None:
        pygame.init()
        set_window_icon()
        pygame.display.set_caption("GoLPy — Conway's Game of Life")
        self.screen = pygame.display.set_mode((720, 640), pygame.RESIZABLE)
        self.clock = pygame.time.Clock()
        self.font = pygame.font.SysFont("Arial", 16)
        self.small_font = pygame.font.SysFont("Arial", 14)

        self.game = GameOfLife(mode)
        self.running_sim = False
        self.show_stats = True
        self.debug = debug
        self.saved: Snapshot | None = None
        self.border_hue = 0.0

        self.cell_size = 10.0
        self.grid_offset_x = 0.0
        self.grid_offset_y = 0.0
        self.zoom_level = 1.0
        self.pan_x = 0.0
        self.pan_y = 0.0
        self._locked_window_size: tuple[int, int] | None = None

        self._pointer_down = False
        self._dragging = False
        self._drag_start = (0, 0)
        self._pan_start = (0.0, 0.0)
        self._last_step_ms = 0.0

        names = pattern_names()
        self.picker = controls.PatternPicker(names)
        self.toolbar = controls.Toolbar(self.screen.get_width())
        self.toolbar.speed.value = speed
        self._configure_zoom_for_mode()
        self._center_view()

        if pattern:
            self._load_pattern(pattern)

    def _play_outer(self) -> pygame.Rect:
        width, height = self.screen.get_size()
        return pygame.Rect(0, controls.Toolbar.HEIGHT, width, height - controls.Toolbar.HEIGHT)

    def _play_inner(self) -> pygame.Rect:
        return self._play_outer().inflate(-2 * FRAME_BORDER, -2 * FRAME_BORDER)

    def _canvas_rect(self) -> pygame.Rect:
        """Grid/playfield area inside the frame border."""
        return self._play_inner()

    def _local_pos(self, pos: tuple[int, int]) -> tuple[float, float]:
        inner = self._play_inner()
        return float(pos[0] - inner.x), float(pos[1] - inner.y)

    def _wrapped_run_locked(self) -> bool:
        return self.game.mode == "wrapped" and self.running_sim

    def _set_window_resizable(self, resizable: bool) -> None:
        size = self.screen.get_size()
        if resizable:
            self.screen = pygame.display.set_mode(size, pygame.RESIZABLE)
            self._locked_window_size = None
        else:
            self.screen = pygame.display.set_mode(size)
            self._locked_window_size = size

    def _lock_window_size(self) -> None:
        self._sync_view_to_canvas()
        self._set_window_resizable(False)

    def _unlock_window_size(self) -> None:
        if self._locked_window_size is not None:
            self._set_window_resizable(True)
            self._sync_view_to_canvas()

    def _sync_view_to_canvas(self) -> None:
        """Fit wrapped grid to the canvas; refresh infinite cell scale."""
        canvas = self._canvas_rect()
        if self.game.mode == "wrapped":
            if not self._wrapped_run_locked():
                cols, rows, cell_size, off_x, off_y = wrapped_grid_layout(
                    canvas.width, canvas.height
                )
                if self.game.set_wrapped_dimensions(cols, rows):
                    self.game.clear()
                    self.picker.last_selected = None
                    self.toolbar.pattern_label = "Pattern.."
                    self.toolbar.buttons["pattern"].label = "Pattern.."
                    self.zoom_level = 1.0
                    self.toolbar.zoom.value = 1.0
                    self.pan_x = 0.0
                    self.pan_y = 0.0
                self.cell_size = cell_size
                self.grid_offset_x = off_x
                self.grid_offset_y = off_y
            if self._wrapped_run_locked():
                self.zoom_level = 1.0
                self.toolbar.zoom.value = 1.0
                self.pan_x = 0.0
                self.pan_y = 0.0
        else:
            side = min(canvas.width, canvas.height) * 0.95
            self.cell_size = side / WRAPPED_MIN_AXIS

    def _cell_sizes(self) -> tuple[float, float]:
        size = self.cell_size * self.zoom_level
        return size, size

    def _configure_zoom_for_mode(self) -> None:
        if self.game.mode == "infinite":
            self.toolbar.zoom.minimum = 10
            self.toolbar.zoom.maximum = 200
            self.toolbar.zoom.step = 1
            self.toolbar.zoom.format_value = "{:.0f}"
            self.toolbar.zoom.value = WRAPPED_MIN_AXIS
            self.toolbar.zoom.enabled = True
            self.zoom_level = 1.0
        else:
            self.toolbar.zoom.minimum = 1
            self.toolbar.zoom.maximum = 4
            self.toolbar.zoom.step = 0.1
            self.toolbar.zoom.format_value = "{:.1f}"
            self.toolbar.zoom.value = 1
            self.zoom_level = 1.0
            self.toolbar.zoom.enabled = not self._wrapped_run_locked()
        self.toolbar.buttons["mode"].label = (
            "Wrapped" if self.game.mode == "wrapped" else "Infinite"
        )
        self._sync_view_to_canvas()

    def _center_view(self) -> None:
        canvas = self._canvas_rect()
        if self.game.mode == "infinite":
            self.pan_x = canvas.width / 2
            self.pan_y = canvas.height / 2
        else:
            self.pan_x = 0.0
            self.pan_y = 0.0

    def _resize_canvas_cell_size(self) -> None:
        self._sync_view_to_canvas()

    def _slider_to_zoom(self) -> float:
        if self.game.mode == "infinite":
            return self.toolbar.zoom.value / WRAPPED_MIN_AXIS
        return self.toolbar.zoom.value

    def _handle_zoom(self, slider_value: float, anchor: tuple[float, float]) -> None:
        if self._wrapped_run_locked():
            return
        zoom_slider = self.toolbar.zoom
        clamped = max(zoom_slider.minimum, min(zoom_slider.maximum, slider_value))
        if self.game.mode == "wrapped":
            if clamped == 1:
                self.zoom_level = 1.0
                self.pan_x = 0.0
                self.pan_y = 0.0
                zoom_slider.value = 1
                return
            cell_x, cell_y = self._cell_sizes()
            world_x = (anchor[0] - self.grid_offset_x - self.pan_x) / cell_x
            world_y = (anchor[1] - self.grid_offset_y - self.pan_y) / cell_y
            self.zoom_level = clamped
            new_x, new_y = self._cell_sizes()
            self.pan_x = anchor[0] - self.grid_offset_x - world_x * new_x
            self.pan_y = anchor[1] - self.grid_offset_y - world_y * new_y
            zoom_slider.value = clamped
            self._clamp_pan()
            return

        cell_x, cell_y = self._cell_sizes()
        if cell_x <= 0 or cell_y <= 0:
            return
        world_x = (anchor[0] - self.pan_x) / cell_x
        world_y = (anchor[1] - self.pan_y) / cell_y
        new_zoom = clamped / WRAPPED_MIN_AXIS
        new_cell = self.cell_size * new_zoom
        self.pan_x = anchor[0] - world_x * new_cell
        self.pan_y = anchor[1] - world_y * new_cell
        self.zoom_level = new_zoom
        zoom_slider.value = clamped

    def _clamp_pan(self) -> None:
        if self.game.mode != "wrapped":
            return
        canvas = self._canvas_rect()
        cell_x, cell_y = self._cell_sizes()
        grid_w = self.game.grid_cols * cell_x
        grid_h = self.game.grid_rows * cell_y
        pan_min_x = -self.grid_offset_x
        pan_max_x = canvas.width - grid_w - self.grid_offset_x
        pan_min_y = -self.grid_offset_y
        pan_max_y = canvas.height - grid_h - self.grid_offset_y
        self.pan_x = min(pan_min_x, max(pan_max_x, self.pan_x))
        self.pan_y = min(pan_min_y, max(pan_max_y, self.pan_y))

    def _viewport_center_cells(self) -> tuple[float, float]:
        canvas = self._canvas_rect()
        cell_x, cell_y = self._cell_sizes()
        x = (canvas.width / 2 - self.grid_offset_x - self.pan_x) / cell_x
        y = (canvas.height / 2 - self.grid_offset_y - self.pan_y) / cell_y
        return x, y

    def _grid_origin(self) -> tuple[float, float]:
        canvas = self._canvas_rect()
        if self.game.mode == "wrapped":
            return (
                canvas.x + self.grid_offset_x + self.pan_x,
                canvas.y + self.grid_offset_y + self.pan_y,
            )
        return canvas.x + self.pan_x, canvas.y + self.pan_y

    def _cell_screen_rect(self, gx: int, gy: int) -> pygame.Rect:
        ox, oy = self._grid_origin()
        cell_x, cell_y = self._cell_sizes()
        left = int(ox + gx * cell_x)
        top = int(oy + gy * cell_y)
        right = int(ox + (gx + 1) * cell_x)
        bottom = int(oy + (gy + 1) * cell_y)
        return pygame.Rect(left, top, max(1, right - left), max(1, bottom - top))

    def _index_at_pixel(
        self, coord: int, origin: float, cell_size: float, cells: int
    ) -> int:
        """Map a screen pixel to a cell index using the same int-snapped bounds as drawing."""
        for i in range(cells):
            if int(origin + i * cell_size) <= coord < int(origin + (i + 1) * cell_size):
                return i
        if coord < int(origin):
            return 0
        return cells - 1

    def _screen_to_cell(self, pos: tuple[int, int]) -> tuple[int, int]:
        ox, oy = self._grid_origin()
        cell_x, cell_y = self._cell_sizes()
        if self.game.mode == "wrapped":
            gx = self._index_at_pixel(pos[0], ox, cell_x, self.game.grid_cols)
            gy = self._index_at_pixel(pos[1], oy, cell_y, self.game.grid_rows)
            return gx, gy

        canvas = self._canvas_rect()
        gx_min = int((canvas.x - ox) / cell_x) - 1
        gx_max = int((canvas.right - ox) / cell_x) + 2
        gy_min = int((canvas.y - oy) / cell_y) - 1
        gy_max = int((canvas.bottom - oy) / cell_y) + 2
        for gy in range(gy_min, gy_max):
            for gx in range(gx_min, gx_max):
                if self._cell_screen_rect(gx, gy).collidepoint(pos):
                    return gx, gy
        return int((pos[0] - ox) / cell_x), int((pos[1] - oy) / cell_y)

    def _load_pattern(self, name: str) -> None:
        if self.game.mode == "wrapped" and not self._wrapped_run_locked():
            self.zoom_level = 1.0
            self.toolbar.zoom.value = 1.0
            self.pan_x = 0.0
            self.pan_y = 0.0
            self._sync_view_to_canvas()
        center = self._viewport_center_cells()
        self.game.load_pattern(name, viewport_center=center)
        self.picker.last_selected = name
        label = next((lbl for key, lbl in self.picker.items if key == name), name)
        self.toolbar.pattern_label = label
        self.toolbar.buttons["pattern"].label = label[:14]

    def _toggle_play(self) -> None:
        if self.running_sim:
            self.running_sim = False
            self.toolbar.buttons["play"].label = "Play"
            if self.game.mode == "wrapped":
                self.toolbar.zoom.enabled = True
                self._unlock_window_size()
        else:
            self.running_sim = True
            self.toolbar.buttons["play"].label = "Pause"
            if self.game.mode == "wrapped":
                self.zoom_level = 1.0
                self.toolbar.zoom.value = 1.0
                self.pan_x = 0.0
                self.pan_y = 0.0
                self.toolbar.zoom.enabled = False
                self._lock_window_size()
            self._last_step_ms = 0.0

    def _step_once(self) -> None:
        if self.running_sim:
            self.running_sim = False
            self.toolbar.buttons["play"].label = "Play"
            if self.game.mode == "wrapped":
                self.toolbar.zoom.enabled = True
                self._unlock_window_size()
        if self.game.mode == "wrapped":
            self.zoom_level = 1.0
            self.toolbar.zoom.value = 1.0
            self.pan_x = 0.0
            self.pan_y = 0.0
            self._sync_view_to_canvas()
        born, died = self.game.step()
        if self.debug:
            print(f"Step: born={born} died={died}", file=sys.stderr)
        self._maybe_debug_log()

    def _reset(self) -> None:
        self.running_sim = False
        self.toolbar.buttons["play"].label = "Play"
        if self.game.mode == "wrapped":
            self._unlock_window_size()
        self.game.clear()
        self.picker.last_selected = None
        self.toolbar.pattern_label = "Pattern.."
        self.toolbar.buttons["pattern"].label = "Pattern.."
        self._configure_zoom_for_mode()
        self._center_view()

    def _toggle_mode(self) -> None:
        new_mode: Mode = "infinite" if self.game.mode == "wrapped" else "wrapped"
        self.running_sim = False
        self.toolbar.buttons["play"].label = "Play"
        if self.game.mode == "wrapped":
            self._unlock_window_size()
        self.game.set_mode(new_mode)
        self.picker.last_selected = None
        self.toolbar.pattern_label = "Pattern.."
        self.toolbar.buttons["pattern"].label = "Pattern.."
        self._resize_canvas_cell_size()
        self._configure_zoom_for_mode()
        self._center_view()

    def _sim_delay_ms(self) -> float:
        speed = max(10, self.toolbar.speed.value)
        return BASE_DELAY_MS * (200 / speed)

    def _maybe_debug_log(self) -> None:
        if not self.debug or self.game.generation % 100 != 0 or not self.game.cells:
            return
        scope, coord = self.game.scope()
        print(
            f"Step: {self.game.generation} Pop: {self.game.population} "
            f"Scope: {scope} {coord}",
            file=sys.stderr,
        )

    def _update_sim(self) -> None:
        if not self.running_sim:
            return
        now = time.monotonic() * 1000
        if self._last_step_ms == 0:
            self._last_step_ms = now
        if now - self._last_step_ms >= self._sim_delay_ms():
            self.game.step()
            self._maybe_debug_log()
            self._last_step_ms = now
            speed_factor = self.toolbar.speed.value / 100
            self.border_hue = (self.border_hue + speed_factor * BORDER_HUE_INCREMENT) % 360

    def _draw_grid(self, surface: pygame.Surface, canvas: pygame.Rect) -> None:
        cell_x, cell_y = self._cell_sizes()
        if cell_x < 1 or cell_y < 1:
            return

        if self.game.mode == "wrapped":
            ox, oy = self._grid_origin()
            right = int(ox + self.game.grid_cols * cell_x)
            bottom = int(oy + self.game.grid_rows * cell_y)
            for col in range(self.game.grid_cols + 1):
                x = int(ox + col * cell_x)
                pygame.draw.line(surface, controls.GRID_COLOR, (x, oy), (x, bottom))
            for row in range(self.game.grid_rows + 1):
                y = int(oy + row * cell_y)
                pygame.draw.line(surface, controls.GRID_COLOR, (ox, y), (right, y))
            return

        view_x_min = (0 - self.pan_x) / cell_x
        view_y_min = (0 - self.pan_y) / cell_y
        view_x_max = (canvas.width - self.pan_x) / cell_x
        view_y_max = (canvas.height - self.pan_y) / cell_y
        gx_min = int(view_x_min) - 1
        gy_min = int(view_y_min) - 1
        gx_max = int(view_x_max) + 2
        gy_max = int(view_y_max) + 2

        for i in range(gx_min, gx_max + 1):
            x = canvas.x + self.pan_x + i * cell_x
            pygame.draw.line(surface, controls.GRID_COLOR, (x, canvas.y), (x, canvas.bottom))
        for j in range(gy_min, gy_max + 1):
            y = canvas.y + self.pan_y + j * cell_y
            pygame.draw.line(surface, controls.GRID_COLOR, (canvas.x, y), (canvas.right, y))

    def _draw_cells(self, surface: pygame.Surface, canvas: pygame.Rect) -> None:
        for (x, y), cell in self.game.cells.items():
            rect = self._cell_screen_rect(x, y)
            if not canvas.colliderect(rect):
                continue
            color = cell_rgb(cell.age, cell.initial_hue)
            pygame.draw.rect(surface, color, rect)

    def _draw_overlays(self, surface: pygame.Surface, canvas: pygame.Rect) -> None:
        if not self.show_stats:
            return
        pop = self.font.render(f"Pop: {self.game.population}", True, controls.TEXT)
        step = self.font.render(f"Step: {self.game.generation}", True, controls.TEXT)
        surface.blit(pop, (canvas.x + 8, canvas.y + 8))
        surface.blit(step, (canvas.right - step.get_width() - 8, canvas.y + 8))

    def draw(self) -> None:
        self.screen.fill(controls.BG)
        self.toolbar.draw(self.screen, self.small_font)
        outer = self._play_outer()
        inner = self._play_inner()
        border_color = self._border_color() if self.running_sim else controls.ACCENT
        pygame.draw.rect(self.screen, border_color, outer)
        pygame.draw.rect(self.screen, controls.PANEL, inner)
        self._draw_grid(self.screen, inner)
        self._draw_cells(self.screen, inner)
        self._draw_overlays(self.screen, inner)
        self.picker.draw(self.screen, self.small_font, self.screen.get_size())
        pygame.display.flip()

    def _border_color(self) -> tuple[int, int, int]:
        from gol.colors import _hsl_to_rgb

        return _hsl_to_rgb(self.border_hue, 100, 65)

    def _on_canvas_click(self, pos: tuple[int, int]) -> None:
        if not self._play_inner().collidepoint(pos):
            return
        gx, gy = self._screen_to_cell(pos)
        self.game.toggle_cell(gx, gy)

    def _can_pan(self) -> bool:
        if self.game.mode == "infinite":
            return True
        return not self.running_sim

    def handle_event(self, event: pygame.event.Event) -> bool:
        if self.picker.visible:
            selected = self.picker.handle_event(event, self.screen.get_size())
            if selected:
                self._load_pattern(selected)
            return True

        if self.toolbar.speed.handle_event(event, self.small_font):
            return True
        if self.toolbar.zoom.handle_event(event, self.small_font):
            if not self._wrapped_run_locked():
                canvas = self._canvas_rect()
                self._handle_zoom(self.toolbar.zoom.value, (canvas.width / 2, canvas.height / 2))
            return True

        if event.type == pygame.VIDEORESIZE:
            if self._wrapped_run_locked() and self._locked_window_size is not None:
                w, h = self._locked_window_size
                self.screen = pygame.display.set_mode((w, h))
                return True
            w = max(MIN_WINDOW[0], event.w)
            h = max(MIN_WINDOW[1], event.h)
            self.screen = pygame.display.set_mode((w, h), pygame.RESIZABLE)
            self.toolbar._layout(w)
            self._sync_view_to_canvas()
            return True

        if event.type == pygame.KEYDOWN:
            if event.key == pygame.K_SPACE:
                self._toggle_play()
                return True
            if event.key == pygame.K_n:
                self._step_once()
                return True
            if event.key == pygame.K_r:
                self._reset()
                return True
            if event.key == pygame.K_h:
                self.show_stats = not self.show_stats
                return True
            if event.key in (pygame.K_PLUS, pygame.K_EQUALS):
                if not self._wrapped_run_locked():
                    z = self.toolbar.zoom
                    inner = self._play_inner()
                    self._handle_zoom(
                        z.value + z.step, (inner.width / 2, inner.height / 2)
                    )
                return True
            if event.key == pygame.K_MINUS:
                if not self._wrapped_run_locked():
                    z = self.toolbar.zoom
                    inner = self._play_inner()
                    self._handle_zoom(
                        z.value - z.step, (inner.width / 2, inner.height / 2)
                    )
                return True

        if event.type == pygame.MOUSEBUTTONDOWN:
            if event.button == 1:
                btn = self.toolbar.hit_button(event.pos)
                if btn == "play":
                    self._toggle_play()
                    return True
                if btn == "step":
                    self._step_once()
                    return True
                if btn == "reset":
                    self._reset()
                    return True
                if btn == "save":
                    self.saved = self.game.snapshot()
                    self.toolbar.buttons["restore"].enabled = True
                    return True
                if btn == "restore":
                    if self.saved:
                        self.game.restore(self.saved)
                    return True
                if btn == "pattern":
                    self.picker.open()
                    return True
                if btn == "mode":
                    self._toggle_mode()
                    return True
                if self._play_inner().collidepoint(event.pos) and self._can_pan():
                    self._pointer_down = True
                    self._dragging = False
                    self._drag_start = event.pos
                    self._pan_start = (self.pan_x, self.pan_y)
                    return True
            elif event.button == 4 and self._play_inner().collidepoint(event.pos):
                if not self._wrapped_run_locked() and self.toolbar.zoom.enabled:
                    z = self.toolbar.zoom
                    self._handle_zoom(z.value + z.step, self._local_pos(event.pos))
                return True
            elif event.button == 5 and self._play_inner().collidepoint(event.pos):
                if not self._wrapped_run_locked() and self.toolbar.zoom.enabled:
                    z = self.toolbar.zoom
                    self._handle_zoom(z.value - z.step, self._local_pos(event.pos))
                return True

        if event.type == pygame.MOUSEMOTION and self._pointer_down:
            dx = event.pos[0] - self._drag_start[0]
            dy = event.pos[1] - self._drag_start[1]
            if abs(dx) > 3 or abs(dy) > 3:
                self._dragging = True
            if self._dragging:
                self.pan_x = self._pan_start[0] + dx
                self.pan_y = self._pan_start[1] + dy
                if self.game.mode == "wrapped":
                    self._clamp_pan()
            return True

        if event.type == pygame.MOUSEBUTTONUP and event.button == 1:
            if self._pointer_down and not self._dragging:
                self._on_canvas_click(event.pos)
            self._pointer_down = False
            self._dragging = False

        return False

    def run(self) -> None:
        self.toolbar.buttons["restore"].enabled = False
        self._sync_view_to_canvas()
        self.zoom_level = self._slider_to_zoom() if self.game.mode == "infinite" else 1.0
        running = True
        while running:
            for event in pygame.event.get():
                if event.type == pygame.QUIT:
                    running = False
                else:
                    self.handle_event(event)
            self._update_sim()
            self.draw()
            self.clock.tick(60)
        pygame.quit()


def run_app(
    *,
    mode: Mode = "wrapped",
    pattern: str | None = None,
    speed: int = 100,
    debug: bool = False,
) -> None:
    GolApp(mode=mode, pattern=pattern, speed=speed, debug=debug).run()
