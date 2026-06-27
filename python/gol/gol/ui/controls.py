"""Toolbar widgets for GoLPy."""

from __future__ import annotations

from dataclasses import dataclass

import pygame

# Theme (gol/index.html CSS variables)
BG = (18, 18, 18)
PANEL = (30, 30, 30)
GRID_COLOR = (51, 51, 51)
ACCENT = (255, 64, 129)
ACCENT2 = (68, 138, 255)
TEXT = (224, 224, 224)
TEXT_MUTED = (85, 85, 85)
DISABLED = (42, 42, 42)


@dataclass
class Rect:
    x: int
    y: int
    w: int
    h: int

    @property
    def pygame_rect(self) -> pygame.Rect:
        return pygame.Rect(self.x, self.y, self.w, self.h)

    def contains(self, pos: tuple[int, int]) -> bool:
        return self.pygame_rect.collidepoint(pos)


class Button:
    def __init__(self, rect: Rect, label: str, *, enabled: bool = True) -> None:
        self.rect = rect
        self.label = label
        self.enabled = enabled

    def draw(self, surface: pygame.Surface, font: pygame.font.Font) -> None:
        color = DISABLED if not self.enabled else PANEL
        border = TEXT_MUTED if not self.enabled else ACCENT2
        pygame.draw.rect(surface, color, self.rect.pygame_rect, border_radius=8)
        pygame.draw.rect(surface, border, self.rect.pygame_rect, 2, border_radius=8)
        text = font.render(self.label, True, TEXT_MUTED if not self.enabled else TEXT)
        tx = self.rect.x + (self.rect.w - text.get_width()) // 2
        ty = self.rect.y + (self.rect.h - text.get_height()) // 2
        surface.blit(text, (tx, ty))

    def hit(self, pos: tuple[int, int]) -> bool:
        return self.enabled and self.rect.contains(pos)


class Slider:
    def __init__(
        self,
        rect: Rect,
        label: str,
        *,
        minimum: float,
        maximum: float,
        value: float,
        step: float = 1.0,
        enabled: bool = True,
        format_value: str = "{:.0f}",
    ) -> None:
        self.rect = rect
        self.label = label
        self.minimum = minimum
        self.maximum = maximum
        self.value = value
        self.step = step
        self.enabled = enabled
        self.format_value = format_value
        self._dragging = False

    def draw(self, surface: pygame.Surface, font: pygame.font.Font) -> None:
        label_text = font.render(
            f"{self.label} {self.format_value.format(self.value)}",
            True,
            TEXT_MUTED if not self.enabled else TEXT,
        )
        surface.blit(label_text, (self.rect.x, self.rect.y))
        track_y = self.rect.y + label_text.get_height() + 4
        track = pygame.Rect(self.rect.x, track_y, self.rect.w, 8)
        pygame.draw.rect(surface, PANEL, track, border_radius=4)
        pygame.draw.rect(surface, TEXT_MUTED if not self.enabled else ACCENT2, track, 1, border_radius=4)
        if self.maximum > self.minimum:
            ratio = (self.value - self.minimum) / (self.maximum - self.minimum)
        else:
            ratio = 0
        knob_x = int(self.rect.x + ratio * (self.rect.w - 12))
        knob = pygame.Rect(knob_x, track_y - 2, 12, 12)
        pygame.draw.rect(surface, ACCENT if self.enabled else TEXT_MUTED, knob, border_radius=6)

    def track_rect(self, font: pygame.font.Font) -> pygame.Rect:
        label_h = font.render(self.label, True, TEXT).get_height()
        return pygame.Rect(self.rect.x, self.rect.y + label_h + 4, self.rect.w, 12)

    def set_from_pos(self, pos: tuple[int, int], font: pygame.font.Font) -> None:
        if not self.enabled:
            return
        track = self.track_rect(font)
        ratio = max(0.0, min(1.0, (pos[0] - track.x) / max(track.w, 1)))
        raw = self.minimum + ratio * (self.maximum - self.minimum)
        if self.step:
            raw = round(raw / self.step) * self.step
        self.value = max(self.minimum, min(self.maximum, raw))

    def handle_event(self, event: pygame.event.Event, font: pygame.font.Font) -> bool:
        if not self.enabled:
            return False
        if event.type == pygame.MOUSEBUTTONDOWN and event.button == 1:
            if self.track_rect(font).collidepoint(event.pos):
                self._dragging = True
                self.set_from_pos(event.pos, font)
                return True
        if event.type == pygame.MOUSEBUTTONUP and event.button == 1:
            if self._dragging:
                self._dragging = False
                return True
        if event.type == pygame.MOUSEMOTION and self._dragging:
            self.set_from_pos(event.pos, font)
            return True
        return False


class PatternPicker:
    def __init__(self, items: list[tuple[str, str]]) -> None:
        self.items = items
        self.visible = False
        self.scroll = 0
        self.row_height = 28
        self.last_selected: str | None = None

    def open(self) -> None:
        self.visible = True
        self.scroll = 0

    def close(self) -> None:
        self.visible = False

    def overlay_rect(self, window_size: tuple[int, int]) -> pygame.Rect:
        w = min(360, window_size[0] - 40)
        h = min(420, window_size[1] - 80)
        return pygame.Rect((window_size[0] - w) // 2, (window_size[1] - h) // 2, w, h)

    def draw(self, surface: pygame.Surface, font: pygame.font.Font, window_size: tuple[int, int]) -> None:
        if not self.visible:
            return
        dim = pygame.Surface(window_size, pygame.SRCALPHA)
        dim.fill((0, 0, 0, 160))
        surface.blit(dim, (0, 0))
        panel = self.overlay_rect(window_size)
        pygame.draw.rect(surface, PANEL, panel, border_radius=8)
        pygame.draw.rect(surface, ACCENT, panel, 2, border_radius=8)
        title = font.render("Select pattern", True, ACCENT)
        surface.blit(title, (panel.x + 12, panel.y + 8))
        list_top = panel.y + 36
        list_height = panel.height - 48
        clip = pygame.Rect(panel.x + 8, list_top, panel.w - 16, list_height)
        surface.set_clip(clip)
        y = list_top - self.scroll
        for key, label in self.items:
            row = pygame.Rect(clip.x, y, clip.w, self.row_height)
            if clip.colliderect(row):
                if key == self.last_selected:
                    pygame.draw.rect(surface, (40, 40, 60), row, border_radius=4)
                text = font.render(label, True, TEXT)
                surface.blit(text, (row.x + 8, row.y + 6))
            y += self.row_height
        surface.set_clip(None)

    def handle_event(self, event: pygame.event.Event, window_size: tuple[int, int]) -> str | None:
        if not self.visible:
            return None
        panel = self.overlay_rect(window_size)
        if event.type == pygame.KEYDOWN and event.key == pygame.K_ESCAPE:
            self.close()
            return None
        if event.type == pygame.MOUSEBUTTONDOWN:
            if event.button == 1:
                list_top = panel.y + 36
                list_height = panel.height - 48
                clip = pygame.Rect(panel.x + 8, list_top, panel.w - 16, list_height)
                if not panel.collidepoint(event.pos):
                    self.close()
                    return None
                if clip.collidepoint(event.pos):
                    index = (event.pos[1] - list_top + self.scroll) // self.row_height
                    if 0 <= index < len(self.items):
                        key, _ = self.items[index]
                        self.last_selected = key
                        self.close()
                        return key
            elif event.button == 4:
                self.scroll = max(0, self.scroll - self.row_height)
            elif event.button == 5:
                max_scroll = max(0, len(self.items) * self.row_height - (panel.height - 48))
                self.scroll = min(max_scroll, self.scroll + self.row_height)
        return None


class Toolbar:
    """Top control bar layout and hit-testing."""

    HEIGHT = 88

    def __init__(self, width: int, pattern_label: str = "Pattern..") -> None:
        self.width = width
        self.pattern_label = pattern_label
        self.buttons: dict[str, Button] = {}
        self.speed = Slider(Rect(0, 0, 140, 36), "Speed", minimum=10, maximum=200, value=100)
        self.zoom = Slider(
            Rect(0, 0, 140, 36),
            "Zoom",
            minimum=1,
            maximum=4,
            value=1,
            step=0.1,
            format_value="{:.1f}",
        )
        self._layout(width)

    def _layout(self, width: int) -> None:
        self.width = width
        y1 = 8
        x = 8
        specs = [
            ("play", "Play"),
            ("step", "Step"),
            ("reset", "Reset"),
            ("save", "M+"),
            ("restore", "MR"),
            ("pattern", self.pattern_label[:14]),
            ("mode", "Wrapped"),
        ]
        self.buttons = {}
        for key, label in specs:
            w = 72 if key not in {"pattern", "mode"} else 110
            self.buttons[key] = Button(Rect(x, y1, w, 32), label)
            x += w + 6
        self.speed.rect = Rect(8, 48, min(160, width // 3), 36)
        self.zoom.rect = Rect(self.speed.rect.x + self.speed.rect.w + 12, 48, min(160, width // 3), 36)

    def draw(self, surface: pygame.Surface, font: pygame.font.Font) -> None:
        bar = pygame.Rect(0, 0, self.width, self.HEIGHT)
        pygame.draw.rect(surface, BG, bar)
        pygame.draw.line(surface, GRID_COLOR, (0, self.HEIGHT - 1), (self.width, self.HEIGHT - 1))
        for button in self.buttons.values():
            button.draw(surface, font)
        self.speed.draw(surface, font)
        self.zoom.draw(surface, font)

    def hit_button(self, pos: tuple[int, int]) -> str | None:
        if pos[1] >= self.HEIGHT:
            return None
        for key, button in self.buttons.items():
            if button.hit(pos):
                return key
        return None
