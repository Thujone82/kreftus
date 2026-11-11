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
	activeRoll      *RollVisualEvent
	rollFrame       int
	aiTimer         int
	shouldQuit      bool
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
}

func (app *GUIApp) Update() error {
	if app.shouldQuit {
		return ebiten.Termination
	}
	app.HandleInput()
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
	app.w = outsideWidth
	app.h = outsideHeight
	return outsideWidth, outsideHeight
}

func RunGUI() error {
	ebiten.SetWindowTitle("Skirmish DICE")
	ebiten.SetWindowSize(1200, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

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
