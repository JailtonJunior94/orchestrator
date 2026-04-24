# Effects Library

Physics-based and visual animation effects available for Bubbletea TUI applications.

## Available Effects

### Metaballs

Lava lamp-style floating blobs using signed distance fields.

```go
type MetaballEffect struct {
    blobs  []blob
    width  int
    height int
    chars  []rune // render charset: e.g. []rune("░▒▓█")
}

type blob struct {
    x, y   float64
    vx, vy float64
    radius float64
}

func (e *MetaballEffect) Update() {
    for i := range e.blobs {
        e.blobs[i].x += e.blobs[i].vx
        e.blobs[i].y += e.blobs[i].vy
        // Bounce on edges
        if e.blobs[i].x <= 0 || e.blobs[i].x >= float64(e.width) {
            e.blobs[i].vx = -e.blobs[i].vx
        }
        if e.blobs[i].y <= 0 || e.blobs[i].y >= float64(e.height) {
            e.blobs[i].vy = -e.blobs[i].vy
        }
    }
}

func (e *MetaballEffect) Render() string {
    // For each cell, sum influence from all blobs
    // influence = radius^2 / ((x-bx)^2 + (y-by)^2)
    // Map total influence to charset index
    // ...
}
```

### Wave Effects

Sine wave distortion applied to text or backgrounds.

```go
type WaveEffect struct {
    amplitude float64
    frequency float64
    phase     float64
    speed     float64
}

func (w *WaveEffect) Tick() {
    w.phase += w.speed
}

// OffsetAt returns vertical offset for column x at tick t.
func (w *WaveEffect) OffsetAt(x int) int {
    return int(w.amplitude * math.Sin(w.frequency*float64(x)+w.phase))
}
```

### Rainbow Cycling

Animated HSL color gradient cycling across text.

```go
// RainbowColor returns an ANSI truecolor escape for position i at time t.
func RainbowColor(i int, t float64) lipgloss.Color {
    hue := math.Mod(float64(i)*0.05+t*0.1, 1.0)
    r, g, b := hslToRGB(hue, 0.8, 0.6)
    return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// ApplyRainbow wraps each rune in s with a cycling color.
func ApplyRainbow(s string, t float64) string {
    var sb strings.Builder
    for i, ch := range s {
        style := lipgloss.NewStyle().Foreground(RainbowColor(i, t))
        sb.WriteString(style.Render(string(ch)))
    }
    return sb.String()
}
```

### Layer Compositor

ANSI-aware multi-layer rendering for overlaying effects.

```go
type Layer struct {
    content string
    zIndex  int
    offsetX int
    offsetY int
}

// Composite merges layers in z-order, respecting transparent cells (space = transparent).
func Composite(width, height int, layers []Layer) string {
    // Sort by zIndex ascending
    // For each cell (x, y), take the topmost non-transparent character
    // Output final grid as joined lines
}
```

## Integration Pattern

Wire effects into the Bubbletea update/view cycle:

```go
// In your model
type Model struct {
    wave    WaveEffect
    rainbow float64 // time accumulator
    tick    int
}

// In Update
case tea.KeyMsg:
    // effects advance on every tick message

case tickMsg:
    m.wave.Tick()
    m.rainbow += 0.05
    m.tick++

// In View
func (m Model) View() string {
    title := ApplyRainbow("My TUI App", m.rainbow)
    // ... render rest of UI
}
```

## Performance Notes

- Metaballs: O(cells × blobs) per frame — keep blob count ≤ 8 for 80×24 terminal
- Wave: O(width) per frame — negligible
- Rainbow: O(len(string)) per frame — avoid on long strings
- All effects: update at ≤ 30fps tick rate to avoid flicker
