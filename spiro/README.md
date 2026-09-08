# SpiroGen — Multi-Node Spirograph Generator v2.7

SpiroGen is an interactive, mathematical pattern generator and Progressive Web App (PWA) built with vanilla HTML5, Canvas 2D, and CSS3. It simulates complex epicycloid, hypotrochoid, and multi-arm harmonic curves produced by connected rotating joints ("nodes").

Live application: [https://kreft.us/spiro/](https://kreft.us/spiro/)

---

## 🌟 Key Features

* **Multi-Node Kinematic Simulation**:
  * Up to 4 chained rotating nodes (base rotor + child arms).
  * Base node supports logarithmic speed scaling (0.1x to 10x), directional control (clockwise, anti-clockwise, fixed), custom starting angles, and total rotation target limits.
  * Child nodes rotate with relative angular speeds proportional to parent revolutions.
  * Individual arm length, trace color, stroke width, opacity (alpha), and drawing toggle per node.

* **HiDPI / Retina Display Scaling**:
  * Automatically detects `window.devicePixelRatio` to scale internal canvas bitmaps to 1:1 physical hardware pixels.
  * Produces razor-sharp lines on Apple Retina displays, high-density mobile screens, and 4K desktop monitors.

* **High-Performance Dual-Canvas Trace Buffer**:
  * Offscreen buffer decouples accumulated curve geometry from real-time animation.
  * Replaces $O(N)$ full-trace redraw loops with $O(1)$ incremental line segments and instant GPU texture blitting.
  * Maintains a locked 60 FPS frame rate even across tens of thousands of continuous revolutions.

* **Mobile-First Responsive Layout**:
  * On viewports $\le 768\text{px}$, the simulation canvas is pinned to the top of the screen in a sticky container.
  * Controls scroll underneath, allowing mobile users to observe patterns emerge in real-time as they adjust parameters.
  * Enlarged slider thumb targets and button heights for touch ergonomics.

* **Dynamic Contrast-Aware Theming**:
  * Preset palettes (*Cosmic Cyan*, *Neon Magenta*, *Electric Lime*, *Golden Hour*) selected randomly on launch.
  * Real-time luminance and YIQ contrast calculations dynamically adjust panel transparency, borders, inputs, and button text colors when background colors change.

* **Snapshot Memory (M+ / MR)**:
  * Save a complete state snapshot (traces, zoom, pan offsets, and color themes) to memory.
  * Recall previous creations instantly without losing your work.

* **Media Exports**:
  * **PNG**: Exports full-resolution, HiDPI-crisp images matching physical device resolution.
  * **GIF**: Generates animated rotation loops using multi-worker NeuQuant color quantization.

* **PWA & Offline Capability**:
  * Fully installable PWA with web app manifest and icons.
  * Offline-capable Service Worker (`sw.js`) with cache-first asset delivery and automated background updates.

* **Easter Egg**:
  * Press and hold the "SpiroGen v2.7" header (or hold M+) to trigger a physics-driven spin-up and decelerating flywheel coast animation.

---

## 📐 Kinematics & Mathematical Model

The curve traced by the tip of node $k$ is calculated via 2D forward kinematics:

$$\begin{aligned}
X_k(t) &= \sum_{i=1}^{k} L_i \cos(\theta_i(t)) \\
Y_k(t) &= \sum_{i=1}^{k} L_i \sin(\theta_i(t))
\end{aligned}$$

Where:
* $L_i$ is the length of arm $i$.
* $\theta_1(t)$ is the absolute angle of the base rotor, driven at angular velocity $\omega_1$:
  $$\theta_1(t + \Delta t) = \theta_1(t) + \omega_1 \cdot \text{direction} \cdot \Delta t$$
* For child nodes $i > 1$, the relative rotation is proportional to parent movement:
  $$\Delta \theta_{i,\text{rel}} = R_i \cdot \Delta \theta_{i-1,\text{abs}}$$
  $$\theta_{i,\text{abs}} = \theta_{i-1,\text{abs}} + \theta_{i,\text{rel}}$$

### Physics Sub-Stepping
To prevent jagged polygons and maintain smooth curves at high rotational velocities, each frame is divided into sub-steps:
* 1–2 nodes: 5 sub-steps per frame
* 3 nodes: 8 sub-steps per frame
* 4 nodes: 11 sub-steps per frame

---

## 📁 Repository Structure

```
spiro/
├── index.html              # Core application DOM structure & meta tags
├── manifest.json           # PWA standalone manifest configuration
├── sw.js                   # Service Worker cache-first offline engine
├── css/
│   └── style.css           # UI theme variables, layout & responsive styling
├── js/
│   ├── app.js              # Simulation engine, kinematics, UI event bindings
│   └── lib/
│       ├── Animated_GIF.js         # Client-side GIF generation wrapper
│       ├── Animated_GIF.worker.js  # Multi-threaded frame processing worker
│       └── NeuQuant.js             # High-quality color quantization
└── icons/                  # PWA application icons (32x32, 192x192, 512x512)
```

---

## 🚀 Local Development & Maintenance

1. **Local Server**:
   Because Service Workers require a secure origin (`localhost` or HTTPS), serve the directory with any local static HTTP server:
   ```bash
   npx serve .
   # or
   python -m http.server 8000
   ```

2. **Cache Updating**:
   When modifying core application files (`index.html`, `style.css`, or `app.js`), update the `CACHE_NAME` version string in [`sw.js`](file:///c:/kreftus/spiro/sw.js#L1):
   ```javascript
   const CACHE_NAME = 'spirograph-generator-v2.7-MMDDYY@HHMM';
   ```
   The application's `controllerchange` listener automatically reloads connected clients when a newly activated Service Worker claims the page.
