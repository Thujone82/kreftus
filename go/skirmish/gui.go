package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type GUIApp struct {
	w, h            int
	screen          ScreenState
	ge              *GameEngine
	headerImg       *ebiten.Image
	showRulesModal  bool
	showExitConfirm bool
	showHandoff     bool
	handoffHoldDraw bool
	ackedHandoffKey string
	autoSelectKey   string
	activeRoll      *RollVisualEvent
	rollFrame       int
	aiTimer         int
	shouldQuit      bool
	toastText       string
	toastFrames     int

	menuBackdrop  *ebiten.Image
	menuBackdropW int
	menuBackdropH int
	tableImg      *ebiten.Image
	tableImgPx    float64
	actionCache   actionSnap
	logCache      logSnap
}

func NewGUIApp() *GUIApp {
	initFonts()

	app := &GUIApp{
		w:         1200,
		h:         800,
		screen:    ScreenMainMenu,
		headerImg: LoadHeaderImage(),
	}

	return app
}

// StartNewGame initializes rosters and begins either quick auto-deployed battle or custom deployment.
func (app *GUIApp) StartNewGame(vsAI bool, quickDeploy bool) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	app.ge = NewGameEngine(roster1, roster2, false, vsAI)
	app.screen = ScreenBattle
	app.showRulesModal = false
	app.showExitConfirm = false
	app.activeRoll = nil
	app.rollFrame = 0
	app.aiTimer = 0
	app.clearBattleUI()

	if quickDeploy {
		// Automatically deploy both armies and jump directly to Round 1 Initiative!
		app.ge.AutoDeployAll()
	}
}

func (app *GUIApp) ReturnToMenu() {
	app.screen = ScreenMainMenu
	app.ge = nil
	app.showRulesModal = false
	app.showExitConfirm = false
	app.activeRoll = nil
	app.rollFrame = 0
	app.aiTimer = 0
	app.clearBattleUI()
}

func (app *GUIApp) clearBattleUI() {
	app.toastText = ""
	app.toastFrames = 0
	app.showHandoff = false
	app.handoffHoldDraw = false
	app.ackedHandoffKey = ""
	app.autoSelectKey = ""
	app.actionCache = actionSnap{}
	app.logCache = logSnap{}
}

func (app *GUIApp) Update() error {
	if app.shouldQuit {
		return ebiten.Termination
	}
	if app.toastFrames > 0 {
		app.toastFrames--
		if app.toastFrames == 0 {
			app.toastText = ""
		}
	}
	app.HandleInput()
	app.showHandoff = app.needsHandoff() || app.handoffHoldDraw
	app.handoffHoldDraw = false
	app.maybeAutoSelect()
	app.ensureActionCache()
	app.updateCursor()
	return nil
}

func (app *GUIApp) Draw(screen *ebiten.Image) {
	if app.screen == ScreenMainMenu {
		app.DrawMainMenu(screen)
	} else {
		app.DrawGame(screen)
	}
}

func (app *GUIApp) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := 1.0
	if m := ebiten.Monitor(); m != nil {
		scale = m.DeviceScaleFactor()
	}
	if scale < 1 {
		scale = 1
	}
	w := int(float64(outsideWidth) * scale)
	h := int(float64(outsideHeight) * scale)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	app.w = w
	app.h = h
	return w, h
}

func RunGUI() error {
	ebiten.SetWindowTitle("Skirmish DICE")
	ebiten.SetWindowSize(1200, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(1100, 720, -1, -1)

	icons := LoadWindowIcons()
	if len(icons) > 0 {
		ebiten.SetWindowIcon(icons)
	}

	app := NewGUIApp()
	err := ebiten.RunGame(app)
	if err == ebiten.Termination {
		return nil
	}
	return err
}
