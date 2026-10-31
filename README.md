# Mini Empires

A small Age of Empires II style isometric RTS in Go, using
[Ebitengine](https://ebitengine.org), with an AI that teaches itself to play
by self-play reinforcement learning.

## Run

```
go run .
```

The main menu offers:

1. **Play vs Scripted AI**: a fixed build order and attack plan.
2. **Play vs Learned AI**: the network trained by self-play.
3. **Watch AI Self-Play**: training starts on all but two CPU cores. The game
   on screen is played by the current network, and it picks up new weights
   on every decision, so you can watch its behaviour change. A dashboard
   shows games played, the win rate against the scripted AI, the mean score,
   policy entropy, and what each side is choosing and with what probability.
4. **Map style**, **Map size** and **Fog of war** (off by default): dropdowns. Fog of war hides
   land you have not explored and dims what you cannot see right now. Sizes run from 32 x 32 to
   256 x 256 tiles, and the default is 96 x 96. Lakes, resources and
   animals scale with the map area. Training uses the chosen size too, and
   training games on big maps get a longer time limit.
5. **Reset Learned AI**: delete the trained network and start again.

The network is saved to `~/.mini-empires/policy.json`. To train without a
window:

```
go run . -train 10m
```

## Rules

- Win by destroying the enemy Town Center.
- **Ages:** Dark, Feudal, Castle, Imperial. Research the next age at the
  Town Center. Advancing needs two different buildings of the current age
  (one Castle to reach the Imperial Age). Each age unlocks buildings and
  units. Military units trained later are stronger, and villagers work faster.
- **Economy:** villagers gather food (berries, hunting, farms), wood, gold and
  stone. They carry resources to the Town Center or to a Mill, Lumber Camp or
  Mining Camp. Farms need a Mill.
- **Animals:** deer run when attacked, and wild boar fight back. Both leave a
  carcass to gather.
- **Buildings:** House, Mill, Lumber Camp, Mining Camp, Farm, Blacksmith,
  Monastery and palisade or stone walls (drag to place a line) on the
  economic page; Barracks, Archery Range, Stable, Siege Workshop, Watch
  Tower and Castle on the military page. Town Centers, towers and castles
  shoot arrows.
- **Units:** Villager, Militia, Spearman (bonus against cavalry), Archer,
  Scout Cavalry, Knight, Cavalry Archer, Longbowman (from the Castle), Monk
  (heals), Battering Ram (destroys buildings), Mangonel (splash damage) and
  Trebuchet (very long range against buildings).
- **Technologies:** economy upgrades at the Town Center, Mill, Lumber Camp
  and Mining Camp (Loom, Wheelbarrow, Double-Bit Axe, Horse Collar and
  more), attack and armor upgrades at the Blacksmith, and unit upgrades at
  the military buildings: Militia → Man-at-Arms → Long Swordsman →
  Champion, Spearman → Pikeman, Archer → Crossbowman → Arbalest, Scout →
  Light Cavalry, Knight → Cavalier. Upgrades apply to units you already
  have.
- **Town Bell:** at the Town Center. Villagers run into the nearest Town
  Center, tower or castle, and each one inside adds an arrow.
- **Score:**
  - Economy: resources gathered / 10, plus the value of completed buildings / 10.
  - Military: army value trained / 10, plus the value of enemies killed / 5,
    plus the value of buildings razed / 5.
  - Technology: 150, 400 or 800 for reaching the Feudal, Castle or Imperial
    Age, plus the value of researched technologies / 10.

## Controls

| Input | Action |
| --- | --- |
| Left click / drag | Select (Shift adds) |
| Right click | Move, gather, hunt, farm, build, drop off, or attack |
| Right drag or middle drag | Pan the map (left drag pans in watch mode) |
| Right click with a building selected | Set its rally point |
| Command panel (bottom left) | Click a button or press its grid hotkey: `Q W E R T / A S D F G / Z X C V B` |
| Villager: `Q` / `W` | Economic / military building page, `B` or `Esc` to go back |
| Town Center: `Q` / `W` | Train a villager / advance to the next age |
| Production queue | Click an entry to cancel it and get a refund |
| Arrow keys, screen edges, Shift+wheel | Scroll |
| Mouse wheel, `PageUp`/`PageDown`, or the `+` `−` buttons | Zoom in and out, from the whole map to 2x |
| `Home` or the Fit button / `Backspace` | Show the whole map / zoom back to 1x |
| `Alt`+`1` to `5` (`Option` on a Mac) | Zoom levels: whole map, overview (the default), close, normal (1x), near (1.6x) |
| Minimap | Left click to jump, right click to send units |
| `Ctrl`+digit / digit | Set / select a control group (press twice to centre on it) |
| `.` | Next idle villager |
| `Delete` | Kill the selected units |
| `Space`, `P` or `F10` | Pause |
| `+` / `-` | Faster / slower game speed (1x to 32x) |
| `F4` | Show or hide the score panel |
| `–` button on a panel | Minimize or restore it (score, command panel, AI panel, training dashboard) |
| `M` / `]` | Music on or off / next track |
| `F11` | Fullscreen |

The toolbar under the top bar has buttons for everything: Slower, the
current speed (click to reset to 1x), Faster (1x to 32x), Pause, music on
or off, Next track and Menu. In play mode it also has Idle villagers and
Scores; in watch mode, Follow fights and New game. The pause and game over
screens have Resume, Play again and Main menu buttons. Keyboard shortcuts:
`+`/`-` change the speed, `Space` (or `P`, `F10`) pauses, `F` follows the
fighting, and `Esc` in watch mode returns to the menu.

## The learning AI

- **Decisions:** every 3 game seconds, the AI picks one of 34 macro actions,
  for example train a villager, build a mill, advance an age, train archers,
  move villagers to gold, or attack. Actions that are not possible are
  masked out. `brain.go` turns each action into unit orders: it picks
  building sites, assigns idle villagers, defends the base and keeps
  attacks going.
- **Inputs:** 96 numbers describing the economy, army, buildings, age,
  nearby resources, the enemy and the score difference.
- **Network:** an actor-critic MLP (64 → 128 → 64) trained with PPO
  (`internal/rl`, pure Go).
- **Reward:** the change in the player's score minus the change in the
  opponent's score, plus ±5 for a win or loss. Training games end after
  20 game minutes, and the higher score wins.
- **Self-play:** both sides use the same network. Every fourth game is
  against the scripted AI, which gives the win-rate curve on the dashboard.
- **Speed:** on an M4 Max, training runs about 10 games per second. In one
  3-minute run, the win rate against the scripted AI rose from about 35% to
  over 90%.

## Art and music

All art and music is generated by code:

- **Sprites:** `cmd/spritegen` renders the PNGs in `assets/` from
  `internal/art`. Regenerate them with `go generate ./...`, and use
  `-sheet <file.png>` to write a contact sheet. Buildings and resources are
  `size * 64` px wide, with the footprint centre at
  `(width / 2, height - size * 16)`. Unit sheets have 8 rows (directions) of
  8 frames: idle, 4 walk, 2 action, dead. Each frame is 48x48 with the feet
  at (24, 42). You can replace any PNG with art of the same layout.
- **Music:** `internal/music` synthesises eight medieval-style tracks, each in
  its own mode, meter and mood (march, jig, dance, battle, pastoral and
  more). They play in a shuffled order, and the next track renders in the
  background. `go run ./cmd/musicgen -all -out <dir>` writes them all as WAV.
- **Ground and water:** a shader draws the whole ground in one pass. It
  blends grass, dry grass, dirt and sand per pixel from `ground_atlas.png`,
  so terrain types fade into each other. Water has animated waves, sun
  glints, depth colour, caustics and shoreline foam.
- **Villagers** are men or women, and are drawn with the tool for their task
  (`villager_<team>_<m|f>_<task>.png`).

## Maps

Every map is generated from its seed. The menu chooses a style (Arabia,
Black Forest, Lakes, Rivers or Oasis), or Random. Base positions,
forests, lakes, rivers and resources all vary. Both players always get a
mirrored, fair start, and a connectivity pass makes sure the bases can
reach each other.

## Layout

| File | Contents |
| --- | --- |
| `defs.go` | Unit, building and age data |
| `world.go`, `mapgen.go` | Terrain, seeded map generation, A* pathfinding, spatial grid |
| `ground.go` | Ground and water shader, terrain blend maps |
| `soundtrack.go` | Music playlist |
| `entities.go` | Unit and building behaviour, combat, hunting, farming |
| `player.go` | Ages, requirements, production queues, scoring |
| `brain.go` | AI actions, observations, masks, the scripted policy |
| `trainer.go` | Self-play workers, PPO learner, saving |
| `game.go` | Game state, tick loop, isometric camera, picking |
| `input.go`, `ui.go` | Player input, command panel |
| `draw.go`, `assets.go` | Rendering, HUD, sprite loading |
| `app.go` | Menu, play and watch modes, training dashboard, music |
| `sim_test.go` | Headless simulation tests |
