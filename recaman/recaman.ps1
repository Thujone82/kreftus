<#
.SYNOPSIS
    Recamán's Sequence Visualizer - High-Performance Terminal Edition
    Recreates the interactive HTML5/Canvas Recamán simulator in the PowerShell command line.

.DESCRIPTION
    Visualizes the mesmerizing Recamán sequence using ultra-high-resolution Braille subpixel
    rasterization, 24-bit TrueColor ANSI rainbow gradients, async audio tone synthesis,
    cinematic camera auto-follow, interactive zoom/pan, speed regulation, and mathematical info.

.PARAMETER Step
    Initial step index to start at (default: 0).

.PARAMETER Speed
    Initial speed slider percentage from 0 to 100 (default: 30).

.PARAMETER Zoom
    Initial zoom magnification factor (default: 5.0).

.PARAMETER Audio
    Enable asynchronous tone synthesis on launch.

.PARAMETER Cinematic
    Enable cinematic camera auto-follow mode on launch.

.PARAMETER Play
    Start animation playback immediately on launch.

.EXAMPLE
    .\recaman.ps1
    Starts the visualizer in interactive mode. Press Space to play.

.EXAMPLE
    .\recaman.ps1 -Play -Cinematic -Audio
    Launches directly in playback mode with auto-follow camera and audio tones enabled.
#>

[CmdletBinding()]
param(
    [Parameter(HelpMessage = "Initial step index")]
    [int]$Step = 0,

    [Parameter(HelpMessage = "Initial animation speed (0-100)")]
    [ValidateRange(0, 100)]
    [int]$Speed = 30,

    [Parameter(HelpMessage = "Initial zoom factor")]
    [double]$Zoom = 5.0,

    [Parameter(HelpMessage = "Enable audio tone synthesis on start")]
    [switch]$Audio,

    [Parameter(HelpMessage = "Enable cinematic camera auto-follow on start")]
    [switch]$Cinematic,

    [Parameter(HelpMessage = "Start playback immediately")]
    [switch]$Play
)

Set-StrictMode -Off

# -----------------------------------------------------------------------------
# 1. High-Performance C# Core Engine (Compiled via Roslyn into current process)
# -----------------------------------------------------------------------------
$csharpTemplate = @"
using System;
using System.Text;
using System.Threading;
using System.Collections.Concurrent;
using System.Collections.Generic;

namespace NAMESPACE_PLACEHOLDER {
    public static class Core {
        public const int LIMIT_BUFFER = 150000;
        public const int TARGET_VALUE = 100000;
        public static readonly int[] Sequence;
        public static readonly uint[] HslPalette;

        static Core() {
            // Precompute HSL (H, 70%, 60%) to 24-bit RGB palette for 360 hues
            HslPalette = new uint[360];
            for (int h = 0; h < 360; h++) {
                HslPalette[h] = HslToRgb(h, 0.70f, 0.60f);
            }

            // Precompute Recamán sequence up to LIMIT_BUFFER
            Sequence = new int[LIMIT_BUFFER + 1];
            var visited = new HashSet<int>(LIMIT_BUFFER);
            visited.Add(0);
            Sequence[0] = 0;
            int current = 0;

            for (int i = 1; i <= LIMIT_BUFFER; i++) {
                int nextLow = current - i;
                int nextHigh = current + i;

                if (nextLow > 0 && !visited.Contains(nextLow)) {
                    current = nextLow;
                } else {
                    current = nextHigh;
                }
                Sequence[i] = current;
                visited.Add(current);
            }
        }

        private static uint HslToRgb(float h, float s, float l) {
            float c = (1f - Math.Abs(2f * l - 1f)) * s;
            float x = c * (1f - Math.Abs((h / 60f) % 2f - 1f));
            float m = l - c / 2f;
            float r1 = 0, g1 = 0, b1 = 0;

            if (h < 60)       { r1 = c; g1 = x; }
            else if (h < 120) { r1 = x; g1 = c; }
            else if (h < 180) { g1 = c; b1 = x; }
            else if (h < 240) { g1 = x; b1 = c; }
            else if (h < 300) { r1 = x; b1 = c; }
            else              { r1 = c; b1 = x; }

            byte r = (byte)Math.Round((r1 + m) * 255f);
            byte g = (byte)Math.Round((g1 + m) * 255f);
            byte b = (byte)Math.Round((b1 + m) * 255f);

            return ((uint)r << 16) | ((uint)g << 8) | b;
        }
    }

    public static class AudioSystem {
        private static readonly ConcurrentQueue<int> _queue = new ConcurrentQueue<int>();
        private static CancellationTokenSource _cts;
        private static Thread _thread;
        private static volatile bool _active = false;

        public static bool IsActive {
            get { return _active; }
        }

        public static void SetActive(bool enable) {
            if (enable) {
                if (_active) return;
                _active = true;
                _cts = new CancellationTokenSource();
                _thread = new Thread(Loop) { IsBackground = true };
                _thread.Start();
            } else {
                _active = false;
                if (_cts != null) _cts.Cancel();
                int dummy;
                while (_queue.TryDequeue(out dummy)) {}
            }
        }

        public static void PlayTone(int value, int stepIndex) {
            if (!_active || stepIndex <= 0) return;

            float frequency;
            const int PEAK_STEP = 10000;
            const int MAX_STEP = 100000;
            const float MIN_FREQ = 80;
            const float PEAK_FREQ = 1200;

            if (stepIndex <= PEAK_STEP) {
                float t = (float)stepIndex / PEAK_STEP;
                frequency = MIN_FREQ + (PEAK_FREQ - MIN_FREQ) * t;
            } else {
                float t = (float)(stepIndex - PEAK_STEP) / (MAX_STEP - PEAK_STEP);
                float safeT = Math.Max(0f, Math.Min(1f, t));
                frequency = PEAK_FREQ - (PEAK_FREQ - MIN_FREQ) * safeT;
            }

            int semitoneMod = (value % 12) * 2;
            frequency += semitoneMod;

            int freqInt = (int)Math.Round(frequency);
            if (freqInt < 37) freqInt = 37;
            if (freqInt > 32767) freqInt = 32767;

            // Maintain small queue to prevent audio lag during fast simulation
            if (_queue.Count < 2) {
                _queue.Enqueue(freqInt);
            }
        }

        private static void Loop() {
            while (_active && !_cts.IsCancellationRequested) {
                int freq;
                if (_queue.TryDequeue(out freq)) {
                    try {
                        Console.Beep(freq, 28);
                    } catch {}
                } else {
                    Thread.Sleep(8);
                }
            }
        }

        public static void Stop() {
            SetActive(false);
        }
    }

    public static class Rasterizer {
        private static byte[] _mask = new byte[0];
        private static uint[] _colors = new uint[0];
        private static char[] _chars = new char[0];
        private static uint[] _charColors = new uint[0];
        private static readonly StringBuilder _sb = new StringBuilder(32768);

        public static string Render(
            int cols, int rows,
            int currentStep,
            float zoom, float offsetX, float offsetY,
            bool isPlaying, bool isCinematic, bool audioEnabled,
            int speedSlider, double rate,
            bool showInfo)
        {
            if (cols < 48) cols = 48;
            if (rows < 16) rows = 16;

            int canvasCols = Math.Max(40, cols - 1);
            int canvasRows = Math.Max(5, rows - 5);
            int totalCells = canvasCols * canvasRows;

            if (_mask.Length < totalCells) {
                _mask = new byte[totalCells];
                _colors = new uint[totalCells];
                _chars = new char[totalCells];
                _charColors = new uint[totalCells];
            } else {
                Array.Clear(_mask, 0, totalCells);
                Array.Clear(_colors, 0, totalCells);
                Array.Clear(_chars, 0, totalCells);
                Array.Clear(_charColors, 0, totalCells);
            }

            int subW = canvasCols * 2;
            int subH = canvasRows * 4;
            int subCenterY = (subH / 2) + (int)Math.Round(offsetY * 4.0f);
            float subStartX = (subW * 0.08f) + (offsetX * 2.0f);
            float unit = zoom * 2.0f;

            // 1. Draw Number Line Baseline (in Braille subpixel row)
            int axisCellY = subCenterY / 4;
            int axisSubRow = subCenterY % 4;
            if (axisSubRow < 0) axisSubRow += 4;

            if (axisCellY >= 0 && axisCellY < canvasRows) {
                int axisMask = (axisSubRow == 0 ? 0x09 : axisSubRow == 1 ? 0x12 : axisSubRow == 2 ? 0x24 : 0xC0);
                for (int x = 0; x < canvasCols; x++) {
                    _mask[axisCellY * canvasCols + x] |= (byte)axisMask;
                    _colors[axisCellY * canvasCols + x] = 0x374151; // gray-700
                }
            }

            // 2. Dynamic Ticks & Numeric Values along Number Line
            if (unit > 0.001f) {
                double targetSpacing = 26.0; // subpixels between numeric ticks
                double desiredStep = targetSpacing / unit;
                double stepVal = 1.0;
                double[] steps = new double[] { 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 50000 };
                foreach (double s in steps) {
                    if (s >= desiredStep) {
                        stepVal = s;
                        break;
                    }
                    stepVal = s;
                }

                int minVal = Math.Max(0, (int)Math.Floor(-subStartX / unit));
                int maxVal = (int)Math.Ceiling((subW - subStartX) / unit);
                int startTick = (int)(Math.Floor(minVal / stepVal) * stepVal);
                if (startTick < 0) startTick = 0;

                for (double v = startTick; v <= maxVal; v += stepVal) {
                    float tickSubX = subStartX + (float)(v * unit);
                    int cellX = (int)Math.Round(tickSubX / 2.0f);
                    if (cellX >= 0 && cellX < canvasCols) {
                        if (axisCellY >= 0 && axisCellY < canvasRows) {
                            _mask[axisCellY * canvasCols + cellX] |= 0xFF;
                            _colors[axisCellY * canvasCols + cellX] = 0x4B5563;
                        }
                        int labelCellY = (axisCellY + 1 < canvasRows) ? (axisCellY + 1) : (axisCellY - 1);
                        if (labelCellY >= 0 && labelCellY < canvasRows) {
                            string lbl = v >= 1000000 ? (v / 1000000.0).ToString("0.#") + "M" :
                                         v >= 1000 ? (v / 1000.0).ToString("0.#") + "k" :
                                         ((int)v).ToString();
                            int lblStart = cellX - lbl.Length / 2;
                            if (lblStart >= 0 && lblStart + lbl.Length < canvasCols) {
                                for (int li = 0; li < lbl.Length; li++) {
                                    int pos = labelCellY * canvasCols + lblStart + li;
                                    _chars[pos] = lbl[li];
                                    _charColors[pos] = 0x6B7280; // muted gray-500
                                }
                            }
                        }
                    }
                }
            }

            // 3. Rasterize Semicircular Arcs with Viewport Culling
            float viewLeft = -100f;
            float viewRight = subW + 100f;
            int maxStep = Math.Min(currentStep, Core.LIMIT_BUFFER);

            for (int i = 1; i <= maxStep; i++) {
                int prev = Core.Sequence[i - 1];
                int curr = Core.Sequence[i];
                float centerVal = (prev + curr) / 2.0f;
                float radiusVal = Math.Abs(curr - prev) / 2.0f;

                float screenCenter = subStartX + centerVal * unit;
                float screenRadius = radiusVal * unit;

                // Viewport culling (matches index.html)
                if ((screenCenter + screenRadius) < viewLeft || (screenCenter - screenRadius) > viewRight) {
                    continue;
                }

                if (screenRadius < 0.5f) {
                    int px = (int)Math.Round(screenCenter);
                    int py = subCenterY;
                    if (px >= 0 && px < subW && py >= 0 && py < subH) {
                        int cx = px / 2;
                        int cy = py / 4;
                        int sx = px % 2;
                        int sy = py % 4;
                        if (sy < 0) sy += 4;
                        int m = (sx == 0) ? (sy == 0 ? 1 : sy == 1 ? 2 : sy == 2 ? 4 : 0x40) : (sy == 0 ? 8 : sy == 1 ? 0x10 : sy == 2 ? 0x20 : 0x80);
                        int idx = cy * canvasCols + cx;
                        _mask[idx] |= (byte)m;
                        _colors[idx] = Core.HslPalette[(i * 2) % 360];
                    }
                    continue;
                }

                bool upper = (i % 2 != 0); // Odd steps above, even steps below
                uint arcColor = Core.HslPalette[(i * 2) % 360];

                int steps = Math.Max(6, (int)(screenRadius * Math.PI * 1.5f));
                for (int k = 0; k <= steps; k++) {
                    double a = Math.PI * k / steps;
                    int px = (int)Math.Round(screenCenter + screenRadius * Math.Cos(a));
                    int py = (int)Math.Round(upper ? (subCenterY - screenRadius * Math.Sin(a)) : (subCenterY + screenRadius * Math.Sin(a)));

                    if (px >= 0 && px < subW && py >= 0 && py < subH) {
                        int cx = px / 2;
                        int cy = py / 4;
                        int sx = px % 2;
                        int sy = py % 4;
                        if (sy < 0) sy += 4;
                        int m = (sx == 0) ? (sy == 0 ? 1 : sy == 1 ? 2 : sy == 2 ? 4 : 0x40) : (sy == 0 ? 8 : sy == 1 ? 0x10 : sy == 2 ? 0x20 : 0x80);
                        int idx = cy * canvasCols + cx;
                        _mask[idx] |= (byte)m;
                        _colors[idx] = arcColor;
                    }
                }
            }

            // 4. Head Indicator & Current Value Tag
            if (currentStep >= 0 && currentStep <= Core.LIMIT_BUFFER) {
                int currentVal = Core.Sequence[currentStep];
                float headSubX = subStartX + currentVal * unit;
                int headCellX = (int)Math.Round(headSubX / 2.0f);
                int headCellY = axisCellY;

                if (headCellX >= 0 && headCellX < canvasCols && headCellY >= 0 && headCellY < canvasRows) {
                    _chars[headCellY * canvasCols + headCellX] = '\u25cf';
                    _charColors[headCellY * canvasCols + headCellX] = 0xFFFFFF; // Pure white head dot

                    // Value label above or below head
                    string valText = currentVal.ToString("N0");
                    int labelRow = (currentStep % 2 != 0) ? (headCellY - 1) : (headCellY + 1);
                    if (labelRow < 0) labelRow = 0;
                    if (labelRow >= canvasRows) labelRow = canvasRows - 1;

                    int valStart = headCellX - valText.Length / 2;
                    if (valStart < 0) valStart = 0;
                    if (valStart + valText.Length >= canvasCols) valStart = canvasCols - valText.Length - 1;

                    for (int vi = 0; vi < valText.Length; vi++) {
                        int p = labelRow * canvasCols + valStart + vi;
                        _chars[p] = valText[vi];
                        _charColors[p] = 0x60A5FA; // Light blue accent (same as HTML)
                    }
                }
            }

            // 5. Build ANSI TrueColor Output Frame with Absolute Row Positioning
            _sb.Clear();
            _sb.Append("\x1b[?7l"); // Disable auto-wrap so lines never wrap/scroll

            // Row 1: Title, Target, Step, and Val
            _sb.Append("\x1b[1;1H");
            _sb.Append("\x1b[1;38;2;96;165;250m \u25c8 RECAM\u00c1N\x1b[0;38;2;168;85;247m'S SEQUENCE\x1b[0m");
            if (cols >= 80) {
                _sb.Append(string.Format("\x1b[38;2;107;114;128m  Target: {0:N0}\x1b[0m", Core.TARGET_VALUE));
            } else {
                _sb.Append("\x1b[38;2;107;114;128m  (100k)\x1b[0m");
            }
            _sb.Append("\x1b[38;2;55;65;81m  \u2502  \x1b[0m");

            int curVal = (currentStep >= 0 && currentStep <= Core.LIMIT_BUFFER) ? Core.Sequence[currentStep] : 0;
            _sb.Append(string.Format("\x1b[38;2;156;163;175mSTEP: \x1b[1;37m{0:N0}\x1b[0m  \x1b[38;2;156;163;175mVAL: \x1b[1;38;2;96;165;250m{1:N0}\x1b[0m\x1b[K\r\n", currentStep, curVal));

            // Row 2: Badges Bar
            _sb.Append("\x1b[2;1H");
            if (isPlaying) {
                _sb.Append(" \x1b[1;97;48;2;22;101;52m \u25b6 PLAY \x1b[0m ");
            } else {
                _sb.Append(" \x1b[1;97;48;2;146;64;14m \u23f8 PAUSE \x1b[0m ");
            }
            if (audioEnabled) {
                _sb.Append("\x1b[1;97;48;2;67;56;202m \u266b AUDIO \x1b[0m ");
            } else {
                _sb.Append("\x1b[38;2;156;163;175;48;2;31;41;55m \u266b MUTE \x1b[0m ");
            }
            if (isCinematic) {
                _sb.Append("\x1b[1;97;48;2;15;118;110m \ud83c\udfa5 AUTO \x1b[0m ");
            } else {
                _sb.Append("\x1b[38;2;156;163;175;48;2;31;41;55m \ud83c\udfa5 MAN \x1b[0m ");
            }
            _sb.Append(string.Format("\x1b[38;2;156;163;175m\u26a1 Spd:\x1b[1;97m{0}%\x1b[0;38;2;107;114;128m(~{1:0.#}/s) ", speedSlider, rate));
            _sb.Append(string.Format("\x1b[38;2;156;163;175m\ud83d\udd0d \x1b[1;97m{0:0.00}x ", zoom));
            _sb.Append(string.Format("\x1b[38;2;156;163;175m\u2194 \x1b[1;97m{0:0.#}\x1b[0m\x1b[K\r\n", offsetX));

            // Row 3: Top Separator Line
            _sb.Append("\x1b[3;1H\x1b[38;2;55;65;81m").Append(new string('\u2500', canvasCols)).Append("\x1b[0m\x1b[K\r\n");

            // Canvas Rows (Rows 4 to rows - 2)
            uint lastColor = 0;
            bool colorSet = false;

            for (int y = 0; y < canvasRows; y++) {
                int screenRow = 4 + y;
                _sb.Append("\x1b[").Append(screenRow).Append(";1H");
                int rowOffset = y * canvasCols;
                for (int x = 0; x < canvasCols; x++) {
                    int idx = rowOffset + x;
                    char ch = _chars[idx];

                    if (ch != 0) {
                        uint c = _charColors[idx];
                        if (!colorSet || c != lastColor) {
                            byte r = (byte)((c >> 16) & 0xFF);
                            byte g = (byte)((c >> 8) & 0xFF);
                            byte b = (byte)(c & 0xFF);
                            _sb.Append("\x1b[1;38;2;").Append(r).Append(';').Append(g).Append(';').Append(b).Append('m');
                            lastColor = c;
                            colorSet = true;
                        }
                        _sb.Append(ch);
                    } else {
                        byte m = _mask[idx];
                        if (m != 0) {
                            uint c = _colors[idx];
                            if (!colorSet || c != lastColor) {
                                byte r = (byte)((c >> 16) & 0xFF);
                                byte g = (byte)((c >> 8) & 0xFF);
                                byte b = (byte)(c & 0xFF);
                                _sb.Append("\x1b[38;2;").Append(r).Append(';').Append(g).Append(';').Append(b).Append('m');
                                lastColor = c;
                                colorSet = true;
                            }
                            _sb.Append((char)(0x2800 + m));
                        } else {
                            if (colorSet) {
                                _sb.Append("\x1b[0m");
                                colorSet = false;
                                lastColor = 0;
                            }
                            _sb.Append(' ');
                        }
                    }
                }
                if (colorSet) {
                    _sb.Append("\x1b[0m");
                    colorSet = false;
                    lastColor = 0;
                }
                _sb.Append("\x1b[K\r\n");
            }

            // Bottom Separator Line (Row rows - 1)
            _sb.Append("\x1b[").Append(rows - 1).Append(";1H\x1b[38;2;55;65;81m").Append(new string('\u2500', canvasCols)).Append("\x1b[0m\x1b[K\r\n");

            // Footer Controls Bar (Row rows - no trailing newline to prevent scrolling)
            _sb.Append("\x1b[").Append(rows).Append(";1H");
            if (cols >= 105) {
                _sb.Append(" \x1b[1;38;2;96;165;250m[Space]\x1b[0;38;2;209;213;219m Play/Pause ");
                _sb.Append("\x1b[1;38;2;96;165;250m[N]\x1b[0;38;2;209;213;219m Step ");
                _sb.Append("\x1b[1;38;2;96;165;250m[R]\x1b[0;38;2;209;213;219m Reset ");
                _sb.Append("\x1b[1;38;2;96;165;250m[C]\x1b[0;38;2;209;213;219m Cam ");
                _sb.Append("\x1b[1;38;2;96;165;250m[A]\x1b[0;38;2;209;213;219m Audio ");
                _sb.Append("\x1b[1;38;2;96;165;250m[+/-]\x1b[0;38;2;209;213;219m Zoom ");
                _sb.Append("\x1b[1;38;2;96;165;250m[\u2191/\u2193]\x1b[0;38;2;209;213;219m Speed ");
                _sb.Append("\x1b[1;38;2;96;165;250m[\u2190/\u2192]\x1b[0;38;2;209;213;219m Pan ");
                _sb.Append("\x1b[1;38;2;96;165;250m[?]\x1b[0;38;2;209;213;219m Info ");
                _sb.Append("\x1b[1;38;2;239;68;68m[Q]\x1b[0;38;2;209;213;219m Quit\x1b[0m\x1b[K");
            } else {
                _sb.Append(" \x1b[1;38;2;96;165;250m[Space]\x1b[0;38;2;209;213;219mPlay ");
                _sb.Append("\x1b[1;38;2;96;165;250m[N]\x1b[0;38;2;209;213;219mStep ");
                _sb.Append("\x1b[1;38;2;96;165;250m[R]\x1b[0;38;2;209;213;219mReset ");
                _sb.Append("\x1b[1;38;2;96;165;250m[C]\x1b[0;38;2;209;213;219mCam ");
                _sb.Append("\x1b[1;38;2;96;165;250m[A]\x1b[0;38;2;209;213;219mSound ");
                _sb.Append("\x1b[1;38;2;96;165;250m[+/-]\x1b[0;38;2;209;213;219mZoom ");
                _sb.Append("\x1b[1;38;2;96;165;250m[?]\x1b[0;38;2;209;213;219mHelp ");
                _sb.Append("\x1b[1;38;2;239;68;68m[Q]\x1b[0;38;2;209;213;219mQuit\x1b[0m\x1b[K");
            }

            // Modal Overlay (if toggled)
            if (showInfo) {
                DrawModalOverlay(_sb, cols, rows);
            }

            return _sb.ToString();
        }

        private static void DrawModalOverlay(StringBuilder sb, int cols, int rows) {
            string[] boxLines = new string[] {
                "\u250c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500 \x1b[1;38;2;168;85;247mThe Recam\u00e1n Sequence\x1b[0;38;2;107;114;128m \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2510",
                "\u2502                                                                    \u2502",
                "\u2502  \x1b[1;37mA recursive sequence defined by a simple rule creating complex    \x1b[0;38;2;107;114;128m\u2502",
                "\u2502  \x1b[1;37mchaotic behavior and mesmerizing semicircular patterns:           \x1b[0;38;2;107;114;128m\u2502",
                "\u2502                                                                    \u2502",
                "\u2502    \x1b[1;38;2;168;85;247ma(0) = 0                                                       \x1b[0;38;2;107;114;128m\u2502",
                "\u2502    \x1b[1;37mFor step \x1b[1;38;2;96;165;250mn\x1b[1;37m:                                                    \x1b[0;38;2;107;114;128m\u2502",
                "\u2502      \x1b[38;2;156;163;175mTry \x1b[1;38;2;74;222;128mprev - n\x1b[0m\x1b[38;2;156;163;175m (jump backwards)                                \x1b[0;38;2;107;114;128m\u2502",
                "\u2502      \x1b[38;2;156;163;175mIf positive & not previously visited: take it!               \x1b[0;38;2;107;114;128m\u2502",
                "\u2502      \x1b[38;2;156;163;175mElse take \x1b[1;38;2;248;113;113mprev + n\x1b[0m\x1b[38;2;156;163;175m (jump forward)                             \x1b[0;38;2;107;114;128m\u2502",
                "\u2502                                                                    \u2502",
                "\u2502  \x1b[3;38;2;156;163;175m\"The sequence that wants to die but can't\" because it             \x1b[0;38;2;107;114;128m\u2502",
                "\u2502  \x1b[3;38;2;156;163;175mconstantly tries to return to zero but is forced outwards.        \x1b[0;38;2;107;114;128m\u2502",
                "\u2502                                                                    \u2502",
                "\u2502 \x1b[0;38;2;55;65;81m\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500 \x1b[1;38;2;96;165;250mInteractive Controls\x1b[0;38;2;55;65;81m \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mSpace\x1b[0m / \x1b[1;38;2;96;165;250mP\x1b[0m         Toggle Play / Pause                           \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mN\x1b[0m / \x1b[1;38;2;96;165;250m.\x1b[0m / \x1b[1;38;2;96;165;250m\u2192\x1b[0m       Single step forward (when paused)             \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mB\x1b[0m / \x1b[1;38;2;96;165;250m,\x1b[0m / \x1b[1;38;2;96;165;250m\u2190\x1b[0m       Single step backward (when paused)            \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mR\x1b[0m                 Reset sequence to step 0                      \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mC\x1b[0m                 Toggle Cinematic Camera (auto-follow arc)     \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mA\x1b[0m / \x1b[1;38;2;96;165;250mM\x1b[0m             Toggle Audio tone synthesis (stereo pitch)    \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250m+\x1b[0m / \x1b[1;38;2;96;165;250m-\x1b[0m (\x1b[1;38;2;96;165;250mZ\x1b[0m / \x1b[1;38;2;96;165;250mX\x1b[0m)     Zoom in / Zoom out                            \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250m\u2191\x1b[0m / \x1b[1;38;2;96;165;250m\u2193\x1b[0m (\x1b[1;38;2;96;165;250m[\x1b[0m / \x1b[1;38;2;96;165;250m]\x1b[0m)     Increase / Decrease animation speed           \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250mA\x1b[0m / \x1b[1;38;2;96;165;250mD\x1b[0m / \x1b[1;38;2;96;165;250mW\x1b[0m / \x1b[1;38;2;96;165;250mS\x1b[0m     Pan number line horizontally / vertically     \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250m0\x1b[0m - \x1b[1;38;2;96;165;250m9\x1b[0m             Quick speed jump (1=10%, 5=50%, 0=100%)       \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;96;165;250m?\x1b[0m / \x1b[1;38;2;96;165;250mH\x1b[0m / \x1b[1;38;2;96;165;250mI\x1b[0m         Toggle this Info card                         \x1b[0;38;2;107;114;128m\u2502",
                "\u2502   \x1b[1;38;2;239;68;68mQ\x1b[0m / \x1b[1;38;2;239;68;68mEsc\x1b[0m           Exit visualizer                               \x1b[0;38;2;107;114;128m\u2502",
                "\u2502                                                                    \u2502",
                "\u2502             \x1b[38;2;156;163;175mPress \x1b[1;38;2;96;165;250m[?]\x1b[0m\x1b[38;2;156;163;175m or \x1b[1;38;2;96;165;250m[Esc]\x1b[0m\x1b[38;2;156;163;175m to return to visualizer             \x1b[0;38;2;107;114;128m\u2502",
                "\u2514\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2518"
            };

            int boxWidth = 70;
            int boxHeight = boxLines.Length;
            int startRow = Math.Max(2, (rows - boxHeight) / 2);
            int startCol = Math.Max(1, (cols - boxWidth) / 2);

            for (int i = 0; i < boxLines.Length; i++) {
                int r = startRow + i;
                if (r >= rows) break;
                sb.Append(string.Format("\x1b[{0};{1}H\x1b[48;2;17;24;39m\x1b[38;2;107;114;128m{2}\x1b[0m", r + 1, startCol + 1, boxLines[i]));
            }
        }
    }
}
"@

# Compile C# types with dynamic content-hash namespace (ensures updates load in existing sessions)
$md5 = [System.Security.Cryptography.MD5]::Create()
$sourceHash = $md5.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($csharpTemplate))
$md5.Dispose()
$tag = "V" + [BitConverter]::ToString($sourceHash).Replace("-", "").Substring(0, 8)
$ns = "RecamanEngine_$tag"
$csharpSource = $csharpTemplate -replace "NAMESPACE_PLACEHOLDER", $ns

if (-not ("$ns.Rasterizer" -as [type])) {
    Add-Type -TypeDefinition $csharpSource
}

$Core = "$ns.Core" -as [type]
$Rasterizer = "$ns.Rasterizer" -as [type]
$AudioSystem = "$ns.AudioSystem" -as [type]

# -----------------------------------------------------------------------------
# 2. Console Initialization & State Setup
# -----------------------------------------------------------------------------
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$prevCursorVisible = $true
try {
    $prevCursorVisible = [Console]::CursorVisible
    [Console]::CursorVisible = $false
} catch {}

# Animation & Navigation State
$currentStep = [Math]::Max(0, [Math]::Min($Core::LIMIT_BUFFER, $Step))
$isPlaying = [bool]$Play
$speedSlider = $Speed
$zoom = [float]$Zoom
$offsetX = 0.0
$offsetY = 0.0
$isCinematic = [bool]$Cinematic
$audioEnabled = [bool]$Audio
$showInfo = $false
$running = $true

if ($audioEnabled) {
    $AudioSystem::SetActive($true)
}

# Speed to rate formula matching index.html
$minRate = 1.0
$maxRate = 140.0
$rate = $minRate * [Math]::Pow($maxRate / $minRate, $speedSlider / 100.0)
$stepIntervalMs = 1000.0 / $rate

# Clear screen once cleanly with ANSI and disable auto-wrap
[Console]::Write("`e[?7l`e[?25l`e[2J`e[H")

$sw = [System.Diagnostics.Stopwatch]::StartNew()
$lastStepTime = $sw.Elapsed.TotalMilliseconds
$lastRenderTime = $sw.Elapsed.TotalMilliseconds
$targetFps = 60.0
$frameIntervalMs = 1000.0 / $targetFps

# Safe key input check
$canCheckKeys = $false
try {
    $canCheckKeys = (-not [Console]::IsInputRedirected)
} catch {}

# -----------------------------------------------------------------------------
# 3. Main Interactive Loop
# -----------------------------------------------------------------------------
try {
    while ($running) {
        $now = $sw.Elapsed.TotalMilliseconds

        # Detect terminal dimensions
        $cols = 100
        $rows = 30
        try {
            if (-not [Console]::IsOutputRedirected -and [Console]::WindowWidth -gt 20) {
                $cols = [Console]::WindowWidth
                $rows = [Console]::WindowHeight
            }
        } catch {}

        # Process user keyboard inputs
        if ($canCheckKeys) {
            while ([Console]::KeyAvailable) {
                $keyInfo = [Console]::ReadKey($true)

                # If Info modal is currently displayed, any key dismisses it
                if ($showInfo) {
                    $showInfo = $false
                    continue
                }

                switch ($keyInfo.Key) {
                    'Spacebar' {
                        $isPlaying = -not $isPlaying
                    }
                    'P' {
                        $isPlaying = -not $isPlaying
                    }
                    'N' {
                        # Single step forward
                        if ($currentStep -lt $Core::LIMIT_BUFFER) {
                            $currentStep++
                            if ($audioEnabled) {
                                $AudioSystem::PlayTone($Core::Sequence[$currentStep], $currentStep)
                            }
                        }
                    }
                    'OemPeriod' {
                        if ($currentStep -lt $Core::LIMIT_BUFFER) {
                            $currentStep++
                            if ($audioEnabled) {
                                $AudioSystem::PlayTone($Core::Sequence[$currentStep], $currentStep)
                            }
                        }
                    }
                    'B' {
                        # Single step backward
                        if ($currentStep -gt 0) {
                            $currentStep--
                        }
                    }
                    'OemComma' {
                        if ($currentStep -gt 0) {
                            $currentStep--
                        }
                    }
                    'R' {
                        # Reset
                        $currentStep = 0
                        $isPlaying = $false
                        $offsetX = 0.0
                        $offsetY = 0.0
                        $isCinematic = $false
                    }
                    'C' {
                        # Toggle Cinematic Mode
                        $isCinematic = -not $isCinematic
                    }
                    'A' {
                        # Toggle Audio
                        $audioEnabled = -not $audioEnabled
                        $AudioSystem::SetActive($audioEnabled)
                    }
                    'M' {
                        # Toggle Audio
                        $audioEnabled = -not $audioEnabled
                        $AudioSystem::SetActive($audioEnabled)
                    }
                    'OemPlus' {
                        # Zoom in
                        $zoom = [Math]::Min(100.0, $zoom * 1.25)
                        $isCinematic = $false
                    }
                    'OemMinus' {
                        # Zoom out
                        $zoom = [Math]::Max(0.01, $zoom / 1.25)
                        $isCinematic = $false
                    }
                    'Z' {
                        $zoom = [Math]::Min(100.0, $zoom * 1.25)
                        $isCinematic = $false
                    }
                    'X' {
                        $zoom = [Math]::Max(0.01, $zoom / 1.25)
                        $isCinematic = $false
                    }
                    'PageUp' {
                        $zoom = [Math]::Min(100.0, $zoom * 1.25)
                        $isCinematic = $false
                    }
                    'PageDown' {
                        $zoom = [Math]::Max(0.01, $zoom / 1.25)
                        $isCinematic = $false
                    }
                    'UpArrow' {
                        $speedSlider = [Math]::Min(100, $speedSlider + 5)
                        $rate = $minRate * [Math]::Pow($maxRate / $minRate, $speedSlider / 100.0)
                        $stepIntervalMs = 1000.0 / $rate
                    }
                    'DownArrow' {
                        $speedSlider = [Math]::Max(0, $speedSlider - 5)
                        $rate = $minRate * [Math]::Pow($maxRate / $minRate, $speedSlider / 100.0)
                        $stepIntervalMs = 1000.0 / $rate
                    }
                    'Oem4' { # Left bracket
                        $speedSlider = [Math]::Max(0, $speedSlider - 5)
                        $rate = $minRate * [Math]::Pow($maxRate / $minRate, $speedSlider / 100.0)
                        $stepIntervalMs = 1000.0 / $rate
                    }
                    'Oem6' { # Right bracket
                        $speedSlider = [Math]::Min(100, $speedSlider + 5)
                        $rate = $minRate * [Math]::Pow($maxRate / $minRate, $speedSlider / 100.0)
                        $stepIntervalMs = 1000.0 / $rate
                    }
                    'LeftArrow' {
                        $offsetX += (15.0 / [Math]::Max(0.1, $zoom))
                        $isCinematic = $false
                    }
                    'RightArrow' {
                        $offsetX -= (15.0 / [Math]::Max(0.1, $zoom))
                        $isCinematic = $false
                    }
                    'A' {
                        $offsetX += (15.0 / [Math]::Max(0.1, $zoom))
                        $isCinematic = $false
                    }
                    'D' {
                        $offsetX -= (15.0 / [Math]::Max(0.1, $zoom))
                        $isCinematic = $false
                    }
                    'W' {
                        $offsetY -= 2.0
                        $isCinematic = $false
                    }
                    'S' {
                        $offsetY += 2.0
                        $isCinematic = $false
                    }
                    'Oem2' { # '?' or '/'
                        $showInfo = -not $showInfo
                    }
                    'H' {
                        $showInfo = -not $showInfo
                    }
                    'I' {
                        $showInfo = -not $showInfo
                    }
                    'Escape' {
                        $running = $false
                    }
                    'Q' {
                        $running = $false
                    }
                    default {
                        # Quick speed jump via number keys 0-9
                        if ($keyInfo.KeyChar -ge '0' -and $keyInfo.KeyChar -le '9') {
                            $digit = [int]($keyInfo.KeyChar - '0')
                            $speedSlider = if ($digit -eq 0) { 100 } else { $digit * 10 }
                            $rate = $minRate * [Math]::Pow($maxRate / $minRate, $speedSlider / 100.0)
                            $stepIntervalMs = 1000.0 / $rate
                        }
                    }
                }
            }
        }

        # Advance step if playing
        if ($isPlaying -and -not $showInfo) {
            $elapsedSinceStep = $now - $lastStepTime
            if ($elapsedSinceStep -ge $stepIntervalMs) {
                $stepsToAdvance = [Math]::Max(1, [int]($elapsedSinceStep / $stepIntervalMs))
                $lastStepTime = $now

                for ($s = 0; $s -lt $stepsToAdvance; $s++) {
                    if ($currentStep -ge $Core::LIMIT_BUFFER -or
                        $Core::Sequence[$currentStep] -ge $Core::TARGET_VALUE) {
                        $isPlaying = $false
                        break
                    }
                    $currentStep++
                    if ($audioEnabled) {
                        $AudioSystem::PlayTone($Core::Sequence[$currentStep], $currentStep)
                    }
                }
            }
        } else {
            $lastStepTime = $now
        }

        # Cinematic Camera Auto-Follow (smooth lerp matching index.html)
        if ($isCinematic -and $currentStep -ge 1) {
            $prevVal = $Core::Sequence[$currentStep - 1]
            $currVal = $Core::Sequence[$currentStep]
            $centerVal = ($prevVal + $currVal) / 2.0
            $arcWidth = [Math]::Abs($currVal - $prevVal)
            $safeArcWidth = [Math]::Max(1.0, [double]$arcWidth)

            $subW = $cols * 2.0
            # Target zoom: Active arc fills ~50% of the canvas subpixel width
            $targetZoom = ($subW * 0.25) / $safeArcWidth
            $targetZoom = [Math]::Max(0.02, [Math]::Min(50.0, $targetZoom))
            $zoom += ($targetZoom - $zoom) * 0.10

            # Target offset: Centers arc center on screen
            $targetOffset = ($subW * 0.2) - ($centerVal * $zoom)
            $offsetX += ($targetOffset - $offsetX) * 0.10
        }

        # Render complete TrueColor frame to terminal
        $frameStr = $Rasterizer::Render(
            $cols, $rows,
            $currentStep,
            $zoom, $offsetX, $offsetY,
            $isPlaying, $isCinematic, $audioEnabled,
            $speedSlider, $rate,
            $showInfo
        )
        [Console]::Write($frameStr)

        # If running non-interactively (input redirected), exit after rendering frame
        if (-not $canCheckKeys) {
            $running = $false
        }

        # Frame pacing
        $frameDuration = $sw.Elapsed.TotalMilliseconds - $now
        $sleepMs = [Math]::Max(4, [int]($frameIntervalMs - $frameDuration))
        Start-Sleep -Milliseconds $sleepMs
    }
}
finally {
    # Clean shutdown: Stop audio worker, restore cursor, re-enable auto-wrap, reset ANSI colors
    $AudioSystem::Stop()
    try {
        [Console]::CursorVisible = $prevCursorVisible
    } catch {}
    [Console]::Write("`e[?7h`e[?25h`e[0m`r`n")
}
