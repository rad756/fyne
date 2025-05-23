package widget

import (
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/internal/widget"
	"fyne.io/fyne/v2/theme"
)

var _ fyne.Draggable = (*Slider)(nil)
var _ fyne.Focusable = (*Slider)(nil)
var _ desktop.Hoverable = (*Slider)(nil)
var _ fyne.Tappable = (*Slider)(nil)
var _ fyne.Disableable = (*Slider)(nil)

// Slider is a widget that can slide between two fixed values.
type Slider struct {
	BaseWidget

	Value float64
	Min   float64
	Max   float64
	Step  float64

	Orientation Orientation
	OnChanged   func(float64) `json:"-"`

	// Since: 2.4
	OnChangeEnded func(float64) `json:"-"`

	binder        basicBinder
	hovered       bool
	focused       bool
	disabled      bool // don't use DisableableWidget so we can put Since comments on funcs
	pendingChange bool // true if value changed since last OnChangeEnded
}

// NewSlider returns a basic slider.
func NewSlider(min, max float64) *Slider {
	slider := &Slider{
		Value:       0,
		Min:         min,
		Max:         max,
		Step:        1,
		Orientation: Horizontal,
	}
	slider.ExtendBaseWidget(slider)
	return slider
}

// NewSliderWithData returns a slider connected with the specified data source.
//
// Since: 2.0
func NewSliderWithData(min, max float64, data binding.Float) *Slider {
	slider := NewSlider(min, max)
	slider.Bind(data)

	return slider
}

// Bind connects the specified data source to this Slider.
// The current value will be displayed and any changes in the data will cause the widget to update.
// User interactions with this Slider will set the value into the data source.
//
// Since: 2.0
func (s *Slider) Bind(data binding.Float) {
	s.binder.SetCallback(s.updateFromData)
	s.binder.Bind(data)

	s.OnChanged = func(_ float64) {
		s.binder.CallWithData(s.writeData)
	}
}

// DragEnd is called when the drag ends.
func (s *Slider) DragEnd() {
	if !s.disabled {
		s.fireChangeEnded()
	}
}

// Dragged is called when a drag event occurs.
func (s *Slider) Dragged(e *fyne.DragEvent) {
	if s.disabled {
		return
	}
	ratio := s.getRatio(&e.PointEvent)
	lastValue := s.Value

	s.updateValue(ratio)
	s.positionChanged(lastValue, s.Value)
}

// Tapped is called when a pointer tapped event is captured.
//
// Since: 2.4
func (s *Slider) Tapped(e *fyne.PointEvent) {
	if s.disabled {
		return
	}

	if !s.focused {
		focusIfNotMobile(s.super())
	}

	ratio := s.getRatio(e)
	lastValue := s.Value

	s.updateValue(ratio)
	s.positionChanged(lastValue, s.Value)
	s.fireChangeEnded()
}

func (s *Slider) positionChanged(lastValue, currentValue float64) {
	if s.almostEqual(lastValue, currentValue) {
		return
	}

	s.Refresh()

	s.pendingChange = true
	if s.OnChanged != nil {
		s.OnChanged(s.Value)
	}
}

func (s *Slider) fireChangeEnded() {
	if !s.pendingChange {
		return
	}
	s.pendingChange = false
	if s.OnChangeEnded != nil {
		s.OnChangeEnded(s.Value)
	}
}

// FocusGained is called when this item gained the focus.
//
// Since: 2.4
func (s *Slider) FocusGained() {
	s.focused = true
	if !s.disabled {
		s.Refresh()
	}
}

// FocusLost is called when this item lost the focus.
//
// Since: 2.4
func (s *Slider) FocusLost() {
	s.focused = false
	if !s.disabled {
		s.Refresh()
	}
}

// MouseIn is called when a desktop pointer enters the widget.
//
// Since: 2.4
func (s *Slider) MouseIn(_ *desktop.MouseEvent) {
	s.hovered = true
	if !s.disabled {
		s.Refresh()
	}
}

// MouseMoved is called when a desktop pointer hovers over the widget.
//
// Since: 2.4
func (s *Slider) MouseMoved(_ *desktop.MouseEvent) {
}

// MouseOut is called when a desktop pointer exits the widget
//
// Since: 2.4
func (s *Slider) MouseOut() {
	s.hovered = false
	if !s.disabled {
		s.Refresh()
	}
}

// TypedKey is called when this item receives a key event.
//
// Since: 2.4
func (s *Slider) TypedKey(key *fyne.KeyEvent) {
	if s.disabled {
		return
	}
	if s.Orientation == Vertical {
		switch key.Name {
		case fyne.KeyUp:
			s.SetValue(s.Value + s.Step)
		case fyne.KeyDown:
			s.SetValue(s.Value - s.Step)
		}
	} else {
		switch key.Name {
		case fyne.KeyLeft:
			s.SetValue(s.Value - s.Step)
		case fyne.KeyRight:
			s.SetValue(s.Value + s.Step)
		}
	}
}

// TypedRune is called when this item receives a char event.
//
// Since: 2.4
func (s *Slider) TypedRune(_ rune) {
}

func (s *Slider) buttonDiameter(inlineIconSize float32) float32 {
	return inlineIconSize - 4 // match radio icons
}

func (s *Slider) endOffset(inlineIconSize, innerPadding float32) float32 {
	return s.buttonDiameter(inlineIconSize)/2 + innerPadding - 1.5 // align with radio icons
}

func (s *Slider) getRatio(e *fyne.PointEvent) float64 {
	th := s.Theme()
	pad := s.endOffset(th.Size(theme.SizeNameInlineIcon), th.Size(theme.SizeNameInnerPadding))

	x := e.Position.X
	y := e.Position.Y
	size := s.Size()

	switch s.Orientation {
	case Vertical:
		if y > size.Height-pad {
			return 0.0
		} else if y < pad {
			return 1.0
		} else {
			return 1 - float64(y-pad)/float64(size.Height-pad*2)
		}
	case Horizontal:
		if x > size.Width-pad {
			return 1.0
		} else if x < pad {
			return 0.0
		} else {
			return float64(x-pad) / float64(size.Width-pad*2)
		}
	}
	return 0.0
}

func (s *Slider) clampValueToRange() {
	if s.Value >= s.Max {
		s.Value = s.Max
		return
	} else if s.Value <= s.Min {
		s.Value = s.Min
		return
	}

	if s.Step == 0 { // extended Slider may not have this set - assume value is not adjusted
		return
	}

	// only work with positive mods so the maths holds up
	value := s.Value
	step := s.Step
	invert := false
	if s.Value < 0 {
		invert = true
		value = -value
	}

	rem := math.Mod(value, step)
	if rem == 0 {
		return
	}
	min := value - rem
	if rem > step/2 {
		min += s.Step
	}

	if invert {
		s.Value = -min
	} else {
		s.Value = min
	}
}

func (s *Slider) updateValue(ratio float64) {
	s.Value = s.Min + ratio*(s.Max-s.Min)

	s.clampValueToRange()
}

// SetValue updates the value of the slider and clamps the value to be within the range.
func (s *Slider) SetValue(value float64) {
	if s.Value == value {
		return
	}

	lastValue := s.Value
	s.Value = value

	s.clampValueToRange()
	s.positionChanged(lastValue, s.Value)
	s.fireChangeEnded()
}

// MinSize returns the size that this widget should not shrink below
func (s *Slider) MinSize() fyne.Size {
	s.ExtendBaseWidget(s)
	return s.BaseWidget.MinSize()
}

// Disable disables the slider
//
// Since: 2.5
func (s *Slider) Disable() {
	if !s.disabled {
		defer s.Refresh()
	}
	s.disabled = true
}

// Enable enables the slider
//
// Since: 2.5
func (s *Slider) Enable() {
	if s.disabled {
		defer s.Refresh()
	}
	s.disabled = false
}

// Disabled returns true if the slider is currently disabled
//
// Since: 2.5
func (s *Slider) Disabled() bool {
	return s.disabled
}

// CreateRenderer links this widget to its renderer.
func (s *Slider) CreateRenderer() fyne.WidgetRenderer {
	s.ExtendBaseWidget(s)
	th := s.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	track := canvas.NewRectangle(th.Color(theme.ColorNameInputBackground, v))
	active := canvas.NewRectangle(th.Color(theme.ColorNameForeground, v))
	thumb := &canvas.Circle{FillColor: th.Color(theme.ColorNameForeground, v)}
	focusIndicator := &canvas.Circle{FillColor: color.Transparent}

	objects := []fyne.CanvasObject{track, active, thumb, focusIndicator}

	slide := &sliderRenderer{widget.NewBaseRenderer(objects), track, active, thumb, focusIndicator, s}
	slide.Refresh() // prepare for first draw
	return slide
}

func (s *Slider) almostEqual(a, b float64) bool {
	delta := math.Abs(a - b)
	return delta <= s.Step/2
}

func (s *Slider) updateFromData(data binding.DataItem) {
	if data == nil {
		return
	}
	floatSource, ok := data.(binding.Float)
	if !ok {
		return
	}

	val, err := floatSource.Get()
	if err != nil {
		fyne.LogError("Error getting current data value", err)
		return
	}
	s.SetValue(val) // if val != s.Value, this will call updateFromData again, but only once
}

func (s *Slider) writeData(data binding.DataItem) {
	if data == nil {
		return
	}
	floatTarget, ok := data.(binding.Float)
	if !ok {
		return
	}
	currentValue, err := floatTarget.Get()
	if err != nil {
		return
	}
	if s.Value != currentValue {
		err := floatTarget.Set(s.Value)
		if err != nil {
			fyne.LogError(fmt.Sprintf("Failed to set binding value to %f", s.Value), err)
		}
	}
}

// Unbind disconnects any configured data source from this Slider.
// The current value will remain at the last value of the data source.
//
// Since: 2.0
func (s *Slider) Unbind() {
	s.OnChanged = nil
	s.binder.Unbind()
}

const minLongSide = float32(34) // added to button diameter

type sliderRenderer struct {
	widget.BaseRenderer
	track          *canvas.Rectangle
	active         *canvas.Rectangle
	thumb          *canvas.Circle
	focusIndicator *canvas.Circle
	slider         *Slider
}

// Refresh updates the widget state for drawing.
func (s *sliderRenderer) Refresh() {
	th := s.slider.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	s.track.FillColor = th.Color(theme.ColorNameInputBackground, v)
	if s.slider.disabled {
		s.thumb.FillColor = th.Color(theme.ColorNameDisabled, v)
	} else {
		s.thumb.FillColor = th.Color(theme.ColorNameForeground, v)
	}
	s.active.FillColor = s.thumb.FillColor

	if s.slider.focused && !s.slider.disabled {
		s.focusIndicator.FillColor = th.Color(theme.ColorNameFocus, v)
	} else if s.slider.hovered && !s.slider.disabled {
		s.focusIndicator.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		s.focusIndicator.FillColor = color.Transparent
	}

	s.focusIndicator.Refresh()

	s.slider.clampValueToRange()
	s.Layout(s.slider.Size())
	canvas.Refresh(s.slider.super())
}

// Layout the components of the widget.
func (s *sliderRenderer) Layout(size fyne.Size) {
	th := s.slider.Theme()

	inputBorderSize := th.Size(theme.SizeNameInputBorder)
	trackWidth := inputBorderSize * 2
	inlineIconSize := th.Size(theme.SizeNameInlineIcon)
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	diameter := s.slider.buttonDiameter(inlineIconSize)
	endPad := s.slider.endOffset(inlineIconSize, innerPadding)

	var trackPos, activePos, thumbPos fyne.Position
	var trackSize, activeSize fyne.Size

	// some calculations are relative to trackSize, so we must update that first
	switch s.slider.Orientation {
	case Vertical:
		trackPos = fyne.NewPos(size.Width/2-inputBorderSize, endPad)
		trackSize = fyne.NewSize(trackWidth, size.Height-endPad*2)

	case Horizontal:
		trackPos = fyne.NewPos(endPad, size.Height/2-inputBorderSize)
		trackSize = fyne.NewSize(size.Width-endPad*2, trackWidth)
	}
	s.track.Move(trackPos)
	s.track.Resize(trackSize)

	activeOffset := s.getOffset(inlineIconSize, innerPadding) // TODO based on old size...0
	switch s.slider.Orientation {
	case Vertical:
		activePos = fyne.NewPos(trackPos.X, activeOffset)
		activeSize = fyne.NewSize(trackWidth, trackSize.Height-activeOffset+endPad)

		thumbPos = fyne.NewPos(
			trackPos.X-(diameter-trackSize.Width)/2, activeOffset-(diameter/2))
	case Horizontal:
		activePos = trackPos
		activeSize = fyne.NewSize(activeOffset-endPad, trackWidth)

		thumbPos = fyne.NewPos(
			activeOffset-(diameter/2), trackPos.Y-(diameter-trackSize.Height)/2)
	}

	s.active.Move(activePos)
	s.active.Resize(activeSize)

	s.thumb.Move(thumbPos)
	s.thumb.Resize(fyne.NewSize(diameter, diameter))

	focusIndicatorSize := fyne.NewSquareSize(inlineIconSize + innerPadding)
	delta := (focusIndicatorSize.Width - diameter) / 2
	s.focusIndicator.Resize(focusIndicatorSize)
	s.focusIndicator.Move(thumbPos.SubtractXY(delta, delta))
}

// MinSize calculates the minimum size of a widget.
func (s *sliderRenderer) MinSize() fyne.Size {
	th := s.slider.Theme()
	pad := th.Size(theme.SizeNameInnerPadding)
	tap := th.Size(theme.SizeNameInlineIcon)
	dia := s.slider.buttonDiameter(tap)
	s1, s2 := minLongSide+dia, tap+pad*2

	switch s.slider.Orientation {
	case Vertical:
		return fyne.NewSize(s2, s1)
	default:
		return fyne.NewSize(s1, s2)
	}
}

func (s *sliderRenderer) getOffset(iconInlineSize, innerPadding float32) float32 {
	endPad := s.slider.endOffset(iconInlineSize, innerPadding)
	w := s.slider
	size := s.track.Size()
	if w.Value == w.Min || w.Min == w.Max {
		switch w.Orientation {
		case Vertical:
			return size.Height + endPad
		case Horizontal:
			return endPad
		}
	}
	ratio := float32((w.Value - w.Min) / (w.Max - w.Min))

	switch w.Orientation {
	case Vertical:
		y := size.Height - ratio*size.Height + endPad
		return y
	case Horizontal:
		x := ratio*size.Width + endPad
		return x
	}

	return endPad
}
