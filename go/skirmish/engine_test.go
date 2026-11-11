package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestFullGameCycle(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, true)

	// Test AutoDeployAll
	ge.AutoDeployAll()
	if len(ge.DeployRemaining0) != 0 || len(ge.DeployRemaining1) != 0 {
		t.Fatalf("expected all units deployed, got remaining0=%d, remaining1=%d",
			len(ge.DeployRemaining0), len(ge.DeployRemaining1))
	}
	if ge.Phase != PhaseTurnAction {
		t.Fatalf("expected PhaseTurnAction after AutoDeployAll, got %v", ge.Phase)
	}
	if ge.ActivationPoints < 2 {
		t.Fatalf("expected at least 2 AP rolled from 2d6, got %d", ge.ActivationPoints)
	}

	// Test AI Turn execution
	ge.Players[ge.ActivePlayerIdx].IsAI = true
	RunAITurn(ge)

	// Test Ending Turn advances turns
	initialTurn := ge.TurnNumber
	ge.EndTurn()
	if ge.TurnNumber != initialTurn+1 {
		t.Fatalf("expected turn to advance to %d, got %d", initialTurn+1, ge.TurnNumber)
	}
}

func TestAITurnCannotAffordActionsEndsTurn(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", false)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, true)
	ge.AutoDeployAll()

	// Force AI player active with only 1 AP, but all AI units have MovesMade = 1 (so next step costs 3 AP)
	aiIdx := 1
	ge.ActivePlayerIdx = aiIdx
	ge.TurnNumber = 1
	ge.ActivationPoints = 1

	for _, u := range ge.GetAliveUnits(aiIdx) {
		u.MovesMade = 1
		u.IsShooter = false
	}

	RunAITurn(ge)

	// AI should have recognized it cannot afford any 3 AP moves, and ended turn to turn 2
	if ge.TurnNumber != 2 {
		t.Fatalf("expected AI to end turn and advance to Turn 2, but turn is %d (AP=%d)",
			ge.TurnNumber, ge.ActivationPoints)
	}
}

func TestGetFaceCachesRoundedSizes(t *testing.T) {
	initFonts()
	a := GetFace(12, FontBold)
	b := GetFace(12, FontBold)
	near := GetFace(12.1, FontBold)
	half := GetFace(12.5, FontBold)
	if a != b || a != near {
		t.Fatal("expected identical faces for sizes that round to the same half pixel")
	}
	if a == half {
		t.Fatal("expected 12.5px to use a distinct face from 12px")
	}
	if a.Size != 12 || half.Size != 12.5 {
		t.Fatalf("unexpected sizes: %v and %v", a.Size, half.Size)
	}
}

func TestMenuChromeBannerHeight(t *testing.T) {
	app := &GUIApp{w: 1200, h: 800}
	chrome := app.menuChrome()
	if chrome.bannerH != 160 || chrome.startY != 254 || chrome.step != 56 {
		t.Fatalf("unexpected menu chrome: %+v", chrome)
	}
	if chrome.btnX != (1200-390)/2 {
		t.Fatalf("button x = %v", chrome.btnX)
	}
}

func TestPassAndPlayHandoffBeforeRoll(t *testing.T) {
	app := NewGUIApp()
	app.w, app.h = 1200, 800
	screen := ebiten.NewImage(app.w, app.h)
	app.Draw(screen)

	app.StartNewGame(false, true)
	if err := app.Update(); err != nil {
		t.Fatal(err)
	}
	if !app.showHandoff {
		t.Fatal("expected the pass-and-play handoff before initiative is shown")
	}
	if app.activeRoll != nil {
		t.Fatal("handoff should hold the roll queue")
	}
	if len(app.ge.RollQueue) == 0 || app.ge.RollQueue[0].Kind != RollVisualInitiative {
		t.Fatalf("expected initiative to stay queued, got %+v", app.ge.RollQueue)
	}
	app.Draw(screen)

	app.ackedHandoffKey = app.handoffKey()
	if err := app.Update(); err != nil {
		t.Fatal(err)
	}
	if app.showHandoff {
		t.Fatal("handoff should clear after the turn is acknowledged")
	}
	if app.activeRoll == nil || app.activeRoll.Kind != RollVisualInitiative {
		t.Fatal("expected the initiative showcase to start after the handoff")
	}

	app.activeRoll = nil
	app.ge.RollQueue = nil
	if err := app.Update(); err != nil {
		t.Fatal(err)
	}
	if app.ge.SelectedUnitID == 0 {
		t.Fatal("expected the human turn to start with a unit selected")
	}

	ai := NewGUIApp()
	ai.StartNewGame(true, true)
	if err := ai.Update(); err != nil {
		t.Fatal(err)
	}
	if ai.showHandoff {
		t.Fatal("AI games should not show the hotseat handoff")
	}
}

func TestLogWrapping(t *testing.T) {
	initFonts()
	face := GetFace(10, FontMono)
	maxWidth := 278.0

	longMsg := "Player 2 (Red) placed Mercenary [d4] (Shooter) at (26.1\", 34.2\")"
	lines := wrapLogText(longMsg, face, maxWidth)

	if len(lines) < 2 {
		t.Fatalf("expected message to wrap into at least 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		w, _ := text.Measure(line, face, 0)
		if w > maxWidth {
			t.Fatalf("line %d [%q] width %.1f exceeds max width %.1f", i, line, w, maxWidth)
		}
	}
}

func TestTurnMovementReset(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", false)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, true)
	ge.AutoDeployAll()

	p0Units := ge.GetFriendlyUnits(0)
	if len(p0Units) == 0 {
		t.Fatalf("no friendly units for player 0")
	}
	testUnit := p0Units[0]

	// Simulate Turn 1: Player 0 moves testUnit 3 times (MovesMade = 3, HasShot = true)
	ge.InitiativeWinner = 0
	ge.StartTurn(1)
	testUnit.MovesMade = 3
	testUnit.HasShot = true

	// Turn 1 ends -> Turn 2 starts (Player 1's turn)
	ge.EndTurn()
	if ge.TurnNumber != 2 {
		t.Fatalf("expected Turn 2, got %d", ge.TurnNumber)
	}

	// Turn 2 ends -> Turn 3 starts (Player 0's next turn in the round)
	ge.EndTurn()
	if ge.TurnNumber != 3 {
		t.Fatalf("expected Turn 3, got %d", ge.TurnNumber)
	}

	// Player 0's testUnit must have MovesMade reset to 0 and HasShot to false
	if testUnit.MovesMade != 0 {
		t.Fatalf("expected testUnit MovesMade to be reset to 0 in Turn 3, got %d", testUnit.MovesMade)
	}
	if testUnit.HasShot {
		t.Fatalf("expected testUnit HasShot to be reset to false in Turn 3")
	}
}

func TestAIMovesForwardAcrossField(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, true)
	ge.AutoDeployAll()

	// Force Player 1 (AI) turn with 6 AP
	ge.InitiativeWinner = 1
	ge.StartTurn(1)
	ge.ActivationPoints = 6

	// Execute full AI turn actions
	for i := 0; i < 10 && ge.ActivePlayerIdx == 1 && ge.Phase == PhaseTurnAction; i++ {
		RunAITurn(ge)
	}

	movedCount := 0
	for _, u := range ge.GetFriendlyUnits(1) {
		if u.MovesMade > 0 {
			movedCount++
		}
	}

	if movedCount < 2 {
		t.Fatalf("expected AI to actively advance at least 2 units forward, only moved %d", movedCount)
	}
}

func TestSwitchSelectionAfterShooting(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, false)
	ge.AutoDeployAll()

	ge.InitiativeWinner = 0
	ge.StartTurn(1)
	ge.ActivationPoints = 5

	friendlies := ge.GetFriendlyUnits(0)
	enemies := ge.GetEnemyUnits(0)
	if len(friendlies) < 2 || len(enemies) < 1 {
		t.Fatalf("insufficient units for test")
	}

	shooter := friendlies[0] // Commander (Shooter)
	nextDie := friendlies[1] // Captain
	target := enemies[0]

	// Position them with guaranteed clear line of sight
	shooter.X, shooter.Y = 5.0, 20.0
	target.X, target.Y = 5.0, 10.0

	if err := ge.SelectUnit(shooter); err != nil {
		t.Fatalf("failed to select shooter: %v", err)
	}
	if err := ge.ShootActiveUnit(target); err != nil {
		t.Fatalf("failed to shoot: %v", err)
	}

	// After shooting, shooter's activation is automatically complete
	if ge.SelectedUnitID != 0 {
		t.Fatalf("expected SelectedUnitID to be cleared after shooting, got %d", ge.SelectedUnitID)
	}

	// Selecting the next die via click must immediately succeed
	if err := ge.SelectUnit(nextDie); err != nil {
		t.Fatalf("expected selecting next die after shooting to succeed, got error: %v", err)
	}
	if ge.SelectedUnitID != nextDie.ID {
		t.Fatalf("expected SelectedUnitID=%d, got %d", nextDie.ID, ge.SelectedUnitID)
	}
}

func TestAutoEndTurnOnZeroAP(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, false)
	ge.AutoDeployAll()

	// Setup Turn 1: Player 0 has 1 AP left, moves a unit
	ge.InitiativeWinner = 0
	ge.StartTurn(1)
	ge.ActivationPoints = 1

	u0 := ge.GetFriendlyUnits(0)[0]
	u0.X, u0.Y = 5.0, 28.0
	if err := ge.SelectUnit(u0); err != nil {
		t.Fatalf("select failed: %v", err)
	}
	if err := ge.MoveActiveUnit(5.0, 24.0); err != nil {
		t.Fatalf("move failed: %v", err)
	}

	// Spending the last AP on a move must automatically advance to Turn 2 (Player 1)
	if ge.TurnNumber != 2 || ge.ActivePlayerIdx != 1 {
		t.Fatalf("expected auto-advance to Turn 2 Player 1 after last AP spent moving, got Turn=%d Player=%d",
			ge.TurnNumber, ge.ActivePlayerIdx)
	}

	// Setup Turn 2: Player 1 has 1 AP left, fires a shot
	ge.ActivationPoints = 1
	shooter := ge.GetFriendlyUnits(1)[0]
	target := ge.GetEnemyUnits(1)[0]
	shooter.X, shooter.Y = 5.0, 5.0
	target.X, target.Y = 5.0, 10.0

	if err := ge.SelectUnit(shooter); err != nil {
		t.Fatalf("select failed: %v", err)
	}
	if err := ge.ShootActiveUnit(target); err != nil {
		t.Fatalf("shoot failed: %v", err)
	}

	// Spending the last AP on a shot must automatically advance to Turn 3 (Player 0)
	if ge.TurnNumber != 3 || ge.ActivePlayerIdx != 0 {
		t.Fatalf("expected auto-advance to Turn 3 Player 0 after last AP spent shooting, got Turn=%d Player=%d",
			ge.TurnNumber, ge.ActivePlayerIdx)
	}
}

func TestUnitBattleStatsTracking(t *testing.T) {
	board := NewBoard()

	// Test cover detection near Center Ruins North (X: 18, Y: 14, Hard Cover +3)
	name, bonus := board.GetNearbyCover(18.0, 14.5)
	if bonus != 3 || name == "Open Ground" {
		t.Fatalf("expected hard cover (+3) near ruins, got %s (+%d)", name, bonus)
	}

	// Test cover detection in open ground (e.g. 2.0, 2.0)
	nameOpen, bonusOpen := board.GetNearbyCover(2.0, 2.0)
	if bonusOpen != 0 || nameOpen != "Open Ground" {
		t.Fatalf("expected Open Ground (+0), got %s (+%d)", nameOpen, bonusOpen)
	}

	// Test combat & recovery stat tracking
	att := &Unit{ID: 1, Type: DieD10, IsShooter: true}
	def := &Unit{ID: 2, Type: DieD4}

	// Force high attack vs low defense: Attacker rolls 9 (+1=10), Defender rolls 0 (+1=1)
	rngHit := &mockRNG{values: []int{9, 0}}
	ResolveShootingCombat(att, def, 0, rngHit)

	if att.ShotsFired != 1 || att.WoundsDealt != 1 || !def.IsWounded {
		t.Fatalf("expected shot stats recorded (shots=%d, wounds=%d, defWounded=%v)",
			att.ShotsFired, att.WoundsDealt, def.IsWounded)
	}
	if att.LastEvent == "" || def.LastEvent == "" {
		t.Fatalf("expected LastEvent populated on both shooter and target")
	}

	// Force melee charge instant kill on wounded defender
	ResolveMeleeCombat(att, def, &mockRNG{})
	if att.Kills != 1 || att.MeleeFights != 1 || def.MeleeFights != 1 {
		t.Fatalf("expected melee kill stats recorded (kills=%d, attFights=%d, defFights=%d)",
			att.Kills, att.MeleeFights, def.MeleeFights)
	}
}

func TestRollShowcaseQueue(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, true)

	// NewGameEngine runs StartDeploymentRoll -> enqueues DeployEdge roll
	if len(ge.RollQueue) == 0 || ge.RollQueue[0].Kind != RollVisualDeployEdge {
		t.Fatalf("expected initial RollVisualDeployEdge queued, got %+v", ge.RollQueue)
	}

	// AutoDeployAll clears DeployEdge and runs StartRoundInitiative + StartTurn(1)
	ge.AutoDeployAll()
	if len(ge.RollQueue) < 2 {
		t.Fatalf("expected Initiative + TurnAP events in RollQueue, got %d", len(ge.RollQueue))
	}
	if ge.RollQueue[0].Kind != RollVisualInitiative {
		t.Fatalf("expected RollVisualInitiative first after AutoDeployAll, got %v", ge.RollQueue[0].Kind)
	}
	if ge.RollQueue[1].Kind != RollVisualTurnAP {
		t.Fatalf("expected RollVisualTurnAP second, got %v", ge.RollQueue[1].Kind)
	}
	if ge.RollQueue[1].RollA1 < 1 || ge.RollQueue[1].RollA2 < 1 {
		t.Fatalf("expected valid 2d6 rolls on TurnAP showcase, got %d, %d",
			ge.RollQueue[1].RollA1, ge.RollQueue[1].RollA2)
	}
}

func TestLoadWindowIconsUsesICO(t *testing.T) {
	icons := LoadWindowIcons()
	if len(icons) != 7 {
		t.Fatalf("expected all 7 frames from skirmish.ico, got %d", len(icons))
	}
	expectedSizes := []int{16, 24, 32, 48, 64, 128, 256}
	for i, exp := range expectedSizes {
		b := icons[i].Bounds()
		if b.Dx() != exp || b.Dy() != exp {
			t.Fatalf("icon frame %d expected %dx%d, got %dx%d", i, exp, exp, b.Dx(), b.Dy())
		}
	}
}

func TestChargeTargetAndInstantKill(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, false)
	ge.AutoDeployAll()

	ge.InitiativeWinner = 0
	ge.StartTurn(1)
	ge.ActivationPoints = 5

	captain := ge.GetFriendlyUnits(0)[1] // Captain d8 (Shooter)
	enemy := ge.GetEnemyUnits(0)[2]      // Soldier d6
	enemy.IsWounded = true               // Showing '1'

	// Place enemy 4.5" away in open ground
	captain.X, captain.Y = 5.0, 20.0
	enemy.X, enemy.Y = 5.0, 15.5

	if err := ge.SelectUnit(captain); err != nil {
		t.Fatalf("failed to select captain: %v", err)
	}
	if err := ge.ChargeTarget(enemy); err != nil {
		t.Fatalf("ChargeTarget failed: %v", err)
	}

	if !enemy.IsCasualty {
		t.Fatalf("expected wounded enemy to be instantly slain in melee charge")
	}
	if captain.Kills != 1 {
		t.Fatalf("expected captain kills=1, got %d", captain.Kills)
	}
	if ge.ActivationPoints != 4 {
		t.Fatalf("expected 1 AP deducted (4 AP left), got %d", ge.ActivationPoints)
	}
}

func TestMoraleAndRecoveryShowcase(t *testing.T) {
	roster1 := NewStandardRoster("Player 1 (Blue)", true)
	roster2 := NewStandardRoster("Player 2 (Red)", false)
	ge := NewGameEngine(roster1, roster2, false, false)
	ge.AutoDeployAll()
	ge.RollQueue = nil

	// Case 1: Failing Morale triggers PhaseGameOver AND still queues RollVisualMorale showcase
	ge.Players[0].Casualties = 1
	ge.Players[1].Casualties = 8
	// d20 rolls: Player 0 rolls 15 (14+1 > 1 -> PASS), Player 1 rolls 5 (4+1 <= 8 -> FAIL)
	ge.RNG = &mockRNG{values: []int{14, 4}}
	ge.StartMoralePhase()

	if ge.Phase != PhaseGameOver || ge.WinnerPlayerID != 0 {
		t.Fatalf("expected PhaseGameOver winner=0 on morale fail, got phase=%v winner=%d", ge.Phase, ge.WinnerPlayerID)
	}
	if len(ge.RollQueue) == 0 || ge.RollQueue[0].Kind != RollVisualMorale {
		t.Fatalf("expected RollVisualMorale queued even on morale failure, got %+v", ge.RollQueue)
	}
	ev := ge.RollQueue[0]
	if !ev.PassedA || ev.PassedB || ev.WinnerID != 0 {
		t.Fatalf("expected PassedA=true PassedB=false WinnerID=0, got PassedA=%v PassedB=%v WinnerID=%d",
			ev.PassedA, ev.PassedB, ev.WinnerID)
	}

	// Case 2: Passing Morale proceeds to Recovery and queues RollVisualRecovery for wounded unit
	ge2 := NewGameEngine(NewStandardRoster("Player 1 (Blue)", true), NewStandardRoster("Player 2 (Red)", false), false, false)
	ge2.AutoDeployAll()
	ge2.RollQueue = nil
	ge2.Players[0].Casualties = 0
	ge2.Players[1].Casualties = 0

	wounded := ge2.GetFriendlyUnits(0)[2] // Soldier d6
	wounded.IsWounded = true
	// Morale: P0 rolls 10 (PASS), P1 rolls 12 (PASS)
	// Recovery: d6 rolls 3 -> 4 (EVEN -> recovered)
	// Next round initiative & Turn 1 rolls follow StartRecoveryPhase: provide extra RNG values
	ge2.RNG = &mockRNG{values: []int{9, 11, 3, 15, 2, 2, 4}}
	ge2.StartMoralePhase()

	if ge2.Phase == PhaseGameOver {
		t.Fatalf("expected battle to continue when both pass morale")
	}
	foundMorale := false
	foundRecovery := false
	for _, item := range ge2.RollQueue {
		if item.Kind == RollVisualMorale {
			foundMorale = true
			if !item.PassedA || !item.PassedB {
				t.Fatalf("expected both PassedA and PassedB true on passing morale")
			}
		}
		if item.Kind == RollVisualRecovery {
			foundRecovery = true
			if item.SidesA != 6 || item.RollA1 != 4 || !item.PassedA {
				t.Fatalf("expected d6 recovery roll=4 (EVEN pass), got sides=%d roll=%d passed=%v",
					item.SidesA, item.RollA1, item.PassedA)
			}
		}
	}
	if !foundMorale || !foundRecovery {
		t.Fatalf("expected both RollVisualMorale (%v) and RollVisualRecovery (%v) in queue", foundMorale, foundRecovery)
	}
}
