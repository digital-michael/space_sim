package commands

import (
	"errors"
	"testing"
)

func TestParse_Empty_ReturnsNil(t *testing.T) {
	cmd, err := Parse("")
	if cmd != nil || err != nil {
		t.Errorf("want nil,nil got %v,%v", cmd, err)
	}
}

func TestParse_Comment_ReturnsNil(t *testing.T) {
	cmd, err := Parse("# this is a comment")
	if cmd != nil || err != nil {
		t.Errorf("want nil,nil got %v,%v", cmd, err)
	}
}

func TestParse_Unknown_ReturnsErrUnknownCommand(t *testing.T) {
	_, err := Parse("frobulate")
	var e ErrUnknownCommand
	if !errors.As(err, &e) {
		t.Errorf("want ErrUnknownCommand, got %T: %v", err, err)
	}
}

func TestParse_SetSpeed_Valid(t *testing.T) {
	cmd, err := Parse("setspeed 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ss, ok := cmd.(SetSpeed)
	if !ok {
		t.Fatalf("want SetSpeed, got %T", cmd)
	}
	if ss.SecondsPerSecond != 10 {
		t.Errorf("want 10, got %v", ss.SecondsPerSecond)
	}
}

func TestParse_SetSpeed_CaseInsensitive(t *testing.T) {
	cmd, err := Parse("SETSPEED 5.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ss, ok := cmd.(SetSpeed)
	if !ok {
		t.Fatalf("want SetSpeed, got %T", cmd)
	}
	if ss.SecondsPerSecond != 5.5 {
		t.Errorf("want 5.5, got %v", ss.SecondsPerSecond)
	}
}

func TestParse_SetSpeed_Zero_Valid(t *testing.T) {
	cmd, err := Parse("setspeed 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ss, ok := cmd.(SetSpeed)
	if !ok {
		t.Fatalf("want SetSpeed, got %T", cmd)
	}
	if ss.SecondsPerSecond != 0 {
		t.Errorf("want 0, got %v", ss.SecondsPerSecond)
	}
}

func TestParse_SetSpeed_Negative_ReturnsErrUsage(t *testing.T) {
	for _, input := range []string{"setspeed -1", "setspeed abc"} {
		_, err := Parse(input)
		var e ErrUsage
		if !errors.As(err, &e) {
			t.Errorf("%q: want ErrUsage, got %T: %v", input, err, err)
		}
	}
}

func TestParse_SetSpeed_MissingArg_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("setspeed")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

func TestParse_GetSpeed(t *testing.T) {
	cmd, err := Parse("getspeed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(GetSpeed); !ok {
		t.Errorf("want GetSpeed, got %T", cmd)
	}
}

func TestParse_SetDataset_AllLevels(t *testing.T) {
	levels := []string{"small", "medium", "large", "huge"}
	for _, level := range levels {
		cmd, err := Parse("setdataset " + level)
		if err != nil {
			t.Fatalf("setdataset %s: unexpected error: %v", level, err)
		}
		sd, ok := cmd.(SetDataset)
		if !ok {
			t.Fatalf("want SetDataset, got %T", cmd)
		}
		if sd.Level != level {
			t.Errorf("level: want %s, got %s", level, sd.Level)
		}
	}
}

func TestParse_SetDataset_InvalidLevel_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("setdataset galactic")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

func TestParse_GetDataset(t *testing.T) {
	cmd, err := Parse("getdataset")
	if err != nil || cmd == nil {
		t.Fatalf("want GetDataset cmd, got err=%v cmd=%v", err, cmd)
	}
	if _, ok := cmd.(GetDataset); !ok {
		t.Errorf("want GetDataset, got %T", cmd)
	}
}

func TestParse_GetTime(t *testing.T) {
	cmd, err := Parse("gettime")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(GetTime); !ok {
		t.Errorf("want GetTime, got %T", cmd)
	}
}

func TestParse_Stream(t *testing.T) {
	cmd, err := Parse("stream")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Stream); !ok {
		t.Errorf("want Stream, got %T", cmd)
	}
}

func TestParse_Help(t *testing.T) {
	cmd, err := Parse("help")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Help); !ok {
		t.Errorf("want Help, got %T", cmd)
	}
}

func TestParse_Quit(t *testing.T) {
	for _, input := range []string{"quit", "exit", "QUIT", "EXIT"} {
		cmd, err := Parse(input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", input, err)
		}
		if _, ok := cmd.(Quit); !ok {
			t.Errorf("%q: want Quit, got %T", input, cmd)
		}
	}
}

func TestParse_Pause(t *testing.T) {
	cmd, err := Parse("pause")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Pause); !ok {
		t.Errorf("want Pause, got %T", cmd)
	}
}

func TestParse_Resume(t *testing.T) {
	cmd, err := Parse("resume")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Resume); !ok {
		t.Errorf("want Resume, got %T", cmd)
	}
}

func TestParse_Bodies_NoFilter(t *testing.T) {
	cmd, err := Parse("bodies")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := cmd.(Bodies)
	if !ok {
		t.Fatalf("want Bodies, got %T", cmd)
	}
	if b.Category != "" {
		t.Errorf("want empty category, got %q", b.Category)
	}
}

func TestParse_Bodies_WithFilter(t *testing.T) {
	cmd, err := Parse("bodies Planet")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := cmd.(Bodies)
	if !ok {
		t.Fatalf("want Bodies, got %T", cmd)
	}
	if b.Category != "planet" {
		t.Errorf("want \"planet\", got %q", b.Category)
	}
}

func TestParse_Bodies_TooManyArgs_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("bodies planet moon")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

func TestParse_Inspect_Valid(t *testing.T) {
	cmd, err := Parse("inspect Earth")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ins, ok := cmd.(Inspect)
	if !ok {
		t.Fatalf("want Inspect, got %T", cmd)
	}
	if ins.Name != "Earth" {
		t.Errorf("want \"Earth\", got %q", ins.Name)
	}
}

func TestParse_Inspect_MultiWord(t *testing.T) {
	cmd, err := Parse("inspect Alpha Centauri A")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ins, ok := cmd.(Inspect)
	if !ok {
		t.Fatalf("want Inspect, got %T", cmd)
	}
	if ins.Name != "Alpha Centauri A" {
		t.Errorf("want \"Alpha Centauri A\", got %q", ins.Name)
	}
}

func TestParse_Inspect_NoArg_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("inspect")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

func TestParse_Status(t *testing.T) {
	cmd, err := Parse("status")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Status); !ok {
		t.Errorf("want Status, got %T", cmd)
	}
}

func TestParse_LeadingTrailingWhitespace(t *testing.T) {
	cmd, err := Parse("  getspeed  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(GetSpeed); !ok {
		t.Errorf("want GetSpeed, got %T", cmd)
	}
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"orbit Earth 15 2", []string{"orbit", "Earth", "15", "2"}},
		{`orbit "S/2019 S 1" 15 2`, []string{"orbit", "S/2019 S 1", "15", "2"}},
		{`nav jump "Ariel's Moon" 2`, []string{"nav", "jump", "Ariel's Moon", "2"}},
		{`camera track "Deep Space Object"`, []string{"camera", "track", "Deep Space Object"}},
		{"getspeed", []string{"getspeed"}},
		{"", nil},
		{`"quoted only"`, []string{"quoted only"}},
		{`a "b c" d`, []string{"a", "b c", "d"}},
		{"  leading  spaces  ", []string{"leading", "spaces"}},
	}
	for _, c := range cases {
		got := tokenize(c.input)
		if len(got) != len(c.want) {
			t.Errorf("tokenize(%q): len=%d want %d: %v", c.input, len(got), len(c.want), got)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("tokenize(%q)[%d]: got %q want %q", c.input, i, got[i], c.want[i])
			}
		}
	}
}

func TestParse_Orbit_QuotedName(t *testing.T) {
	cmd, err := Parse(`orbit "S/2019 S 1" 15 2`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	o, ok := cmd.(Orbit)
	if !ok {
		t.Fatalf("want Orbit, got %T", cmd)
	}
	if o.Name != "S/2019 S 1" {
		t.Errorf("want name %q, got %q", "S/2019 S 1", o.Name)
	}
	if o.SpeedDegPerSec != 15 {
		t.Errorf("want speed 15, got %v", o.SpeedDegPerSec)
	}
	if o.Orbits != 2 {
		t.Errorf("want orbits 2, got %v", o.Orbits)
	}
}

func TestParse_Track_BodyName(t *testing.T) {
	cmd, err := Parse("track Earth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := cmd.(CameraTrack)
	if !ok {
		t.Fatalf("want CameraTrack, got %T", cmd)
	}
	if c.Name != "Earth" {
		t.Errorf("want name %q, got %q", "Earth", c.Name)
	}
}

func TestParse_Track_Stop(t *testing.T) {
	cmd, err := Parse("track stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := cmd.(CameraTrack)
	if !ok {
		t.Fatalf("want CameraTrack, got %T", cmd)
	}
	if c.Name != "" {
		t.Errorf("want empty name (free-fly), got %q", c.Name)
	}
}

func TestParse_Track_StopCaseInsensitive(t *testing.T) {
	cmd, err := Parse("track STOP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := cmd.(CameraTrack)
	if !ok {
		t.Fatalf("want CameraTrack, got %T", cmd)
	}
	if c.Name != "" {
		t.Errorf("want empty name (free-fly), got %q", c.Name)
	}
}

func TestParse_Track_NoArgs_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("track")
	if err == nil {
		t.Fatal("expected error for missing argument, got nil")
	}
}

func TestParse_Labels_On(t *testing.T) {
	cmd, err := Parse("labels on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, ok := cmd.(Labels)
	if !ok {
		t.Fatalf("want Labels, got %T", cmd)
	}
	if l.Mode != "on" {
		t.Errorf("want mode %q, got %q", "on", l.Mode)
	}
}

func TestParse_Labels_Off(t *testing.T) {
	cmd, err := Parse("labels off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, ok := cmd.(Labels)
	if !ok {
		t.Fatalf("want Labels, got %T", cmd)
	}
	if l.Mode != "off" {
		t.Errorf("want mode %q, got %q", "off", l.Mode)
	}
}

func TestParse_Labels_Nearest(t *testing.T) {
	cmd, err := Parse("labels nearest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, ok := cmd.(Labels)
	if !ok {
		t.Fatalf("want Labels, got %T", cmd)
	}
	if l.Mode != "nearest" {
		t.Errorf("want mode %q, got %q", "nearest", l.Mode)
	}
}

func TestParse_Labels_AliasLabel(t *testing.T) {
	cmd, err := Parse("label nearest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, ok := cmd.(Labels)
	if !ok {
		t.Fatalf("want Labels, got %T", cmd)
	}
	if l.Mode != "nearest" {
		t.Errorf("want mode %q, got %q", "nearest", l.Mode)
	}
}

func TestParse_Labels_UnknownMode_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("labels blinking")
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
}

// ─── System ───────────────────────────────────────────────────────────────────

func TestParse_System_List(t *testing.T) {
	cmd, err := Parse("system list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(SystemList); !ok {
		t.Errorf("want SystemList, got %T", cmd)
	}
}

func TestParse_System_Get(t *testing.T) {
	cmd, err := Parse("system get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(SystemGet); !ok {
		t.Errorf("want SystemGet, got %T", cmd)
	}
}

func TestParse_System_Load(t *testing.T) {
	cmd, err := Parse("system load solar_system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sl, ok := cmd.(SystemLoad)
	if !ok {
		t.Fatalf("want SystemLoad, got %T", cmd)
	}
	if sl.Label != "solar_system" {
		t.Errorf("want solar_system, got %q", sl.Label)
	}
}

func TestParse_System_Load_NoLabel_ReturnsError(t *testing.T) {
	_, err := Parse("system load")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestParse_System_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("system")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestParse_System_UnknownSub_ReturnsError(t *testing.T) {
	_, err := Parse("system bogus")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Window ───────────────────────────────────────────────────────────────────

func TestParse_Window_Get(t *testing.T) {
	cmd, err := Parse("window get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(WindowGet); !ok {
		t.Errorf("want WindowGet, got %T", cmd)
	}
}

func TestParse_Window_Maximize(t *testing.T) {
	cmd, err := Parse("window maximize")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(WindowMaximize); !ok {
		t.Errorf("want WindowMaximize, got %T", cmd)
	}
}

func TestParse_Window_Restore(t *testing.T) {
	cmd, err := Parse("window restore")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(WindowRestore); !ok {
		t.Errorf("want WindowRestore, got %T", cmd)
	}
}

func TestParse_Window_Size_Valid(t *testing.T) {
	cmd, err := Parse("window size 1920x1080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ws, ok := cmd.(WindowSize)
	if !ok {
		t.Fatalf("want WindowSize, got %T", cmd)
	}
	if ws.Width != 1920 || ws.Height != 1080 {
		t.Errorf("want 1920x1080, got %dx%d", ws.Width, ws.Height)
	}
}

func TestParse_Window_Size_BadFormat_ReturnsError(t *testing.T) {
	for _, input := range []string{"window size 1920", "window size axb", "window size 0x1080"} {
		_, err := Parse(input)
		if err == nil {
			t.Errorf("%q: want error, got nil", input)
		}
	}
}

func TestParse_Window_Full_On(t *testing.T) {
	cmd, err := Parse("window full on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wf, ok := cmd.(WindowFullscreen)
	if !ok {
		t.Fatalf("want WindowFullscreen, got %T", cmd)
	}
	if !wf.On {
		t.Error("want On=true")
	}
}

func TestParse_Window_Full_Off(t *testing.T) {
	cmd, err := Parse("window full off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wf, ok := cmd.(WindowFullscreen)
	if !ok {
		t.Fatalf("want WindowFullscreen, got %T", cmd)
	}
	if wf.On {
		t.Error("want On=false")
	}
}

func TestParse_Window_Full_NoArg_ReturnsError(t *testing.T) {
	_, err := Parse("window full")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestParse_Window_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("window")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Camera ───────────────────────────────────────────────────────────────────

func TestParse_Camera_Get(t *testing.T) {
	cmd, err := Parse("camera get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(CameraGet); !ok {
		t.Errorf("want CameraGet, got %T", cmd)
	}
}

func TestParse_Camera_Center(t *testing.T) {
	cmd, err := Parse("camera center")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(CameraCenter); !ok {
		t.Errorf("want CameraCenter, got %T", cmd)
	}
}

func TestParse_Camera_Orient_Valid(t *testing.T) {
	cmd, err := Parse("camera orient 45.0 -10.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	co, ok := cmd.(CameraOrient)
	if !ok {
		t.Fatalf("want CameraOrient, got %T", cmd)
	}
	if co.YawDeg != 45.0 || co.PitchDeg != -10.5 {
		t.Errorf("want yaw=45 pitch=-10.5, got yaw=%v pitch=%v", co.YawDeg, co.PitchDeg)
	}
}

func TestParse_Camera_Orient_MissingArgs_ReturnsError(t *testing.T) {
	_, err := Parse("camera orient 45")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestParse_Camera_Position_Valid(t *testing.T) {
	cmd, err := Parse("camera position 1.0 2.0 3.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cp, ok := cmd.(CameraPosition)
	if !ok {
		t.Fatalf("want CameraPosition, got %T", cmd)
	}
	if cp.X != 1.0 || cp.Y != 2.0 || cp.Z != 3.0 {
		t.Errorf("want 1/2/3, got %v/%v/%v", cp.X, cp.Y, cp.Z)
	}
}

func TestParse_Camera_Track_Named(t *testing.T) {
	cmd, err := Parse("camera track Earth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ct, ok := cmd.(CameraTrack)
	if !ok {
		t.Fatalf("want CameraTrack, got %T", cmd)
	}
	if ct.Name != "Earth" {
		t.Errorf("want Earth, got %q", ct.Name)
	}
}

func TestParse_Camera_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("camera")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Nav ──────────────────────────────────────────────────────────────────────

func TestParse_Nav_Stop(t *testing.T) {
	cmd, err := Parse("nav stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(NavStop); !ok {
		t.Errorf("want NavStop, got %T", cmd)
	}
}

func TestParse_Nav_Velocity(t *testing.T) {
	cmd, err := Parse("nav velocity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(NavVelocity); !ok {
		t.Errorf("want NavVelocity, got %T", cmd)
	}
}

func TestParse_Nav_Move_Forward(t *testing.T) {
	cmd, err := Parse("nav forward 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nm, ok := cmd.(NavMove)
	if !ok {
		t.Fatalf("want NavMove, got %T", cmd)
	}
	if nm.Dir != "forward" || nm.Velocity != 100 {
		t.Errorf("want dir=forward vel=100, got dir=%q vel=%v", nm.Dir, nm.Velocity)
	}
}

func TestParse_Nav_Move_AllDirections(t *testing.T) {
	for _, dir := range []string{"back", "left", "right", "up", "down"} {
		cmd, err := Parse("nav " + dir + " 50")
		if err != nil {
			t.Fatalf("nav %s: unexpected error: %v", dir, err)
		}
		nm, ok := cmd.(NavMove)
		if !ok || nm.Dir != dir {
			t.Errorf("nav %s: want NavMove{Dir:%s}, got %T", dir, dir, cmd)
		}
	}
}

func TestParse_Nav_Jump_Single(t *testing.T) {
	cmd, err := Parse("nav jump Earth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nj, ok := cmd.(NavJump)
	if !ok {
		t.Fatalf("want NavJump, got %T", cmd)
	}
	if len(nj.Names) != 1 || nj.Names[0] != "Earth" {
		t.Errorf("want [Earth], got %v", nj.Names)
	}
}

func TestParse_Nav_Jump_Clear(t *testing.T) {
	cmd, err := Parse("nav jump clear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ct, ok := cmd.(CameraTrack)
	if !ok {
		t.Fatalf("want CameraTrack, got %T", cmd)
	}
	if ct.Name != "" {
		t.Errorf("want empty name, got %q", ct.Name)
	}
}

func TestParse_Nav_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("nav")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Perf ─────────────────────────────────────────────────────────────────────

func TestParse_Perf_Get(t *testing.T) {
	cmd, err := Parse("perf get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(PerfGet); !ok {
		t.Errorf("want PerfGet, got %T", cmd)
	}
}

func TestParse_Perf_Set(t *testing.T) {
	cmd, err := Parse("perf set fps 60")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ps, ok := cmd.(PerfSet)
	if !ok {
		t.Fatalf("want PerfSet, got %T", cmd)
	}
	if ps.Field != "fps" || ps.Value != "60" {
		t.Errorf("want fps/60, got %q/%q", ps.Field, ps.Value)
	}
}

func TestParse_Perf_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("perf")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Sleep ────────────────────────────────────────────────────────────────────

func TestParse_Sleep_Valid(t *testing.T) {
	cmd, err := Parse("sleep 2.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := cmd.(Sleep)
	if !ok {
		t.Fatalf("want Sleep, got %T", cmd)
	}
	if s.Seconds != 2.5 {
		t.Errorf("want 2.5, got %v", s.Seconds)
	}
}

func TestParse_Sleep_BadArg_ReturnsErrUsage(t *testing.T) {
	for _, input := range []string{"sleep", "sleep abc", "sleep -1"} {
		_, err := Parse(input)
		var e ErrUsage
		if !errors.As(err, &e) {
			t.Errorf("%q: want ErrUsage, got %T: %v", input, err, err)
		}
	}
}

// ─── Infra ────────────────────────────────────────────────────────────────────

func TestParse_Infra_Modes(t *testing.T) {
	for _, tc := range []struct {
		input string
		mode  int
	}{
		{"infra 0", 0},
		{"infra 1", 1},
		{"infra 2", 2},
	} {
		cmd, err := Parse(tc.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.input, err)
		}
		inf, ok := cmd.(Infra)
		if !ok {
			t.Fatalf("%q: want Infra, got %T", tc.input, cmd)
		}
		if inf.Mode != tc.mode {
			t.Errorf("%q: want mode %d, got %d", tc.input, tc.mode, inf.Mode)
		}
	}
}

func TestParse_Infra_BadMode_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("infra 3")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

// ─── Sync ─────────────────────────────────────────────────────────────────────

func TestParse_Sync_On(t *testing.T) {
	cmd, err := Parse("sync on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := cmd.(Sync)
	if !ok {
		t.Fatalf("want Sync, got %T", cmd)
	}
	if !s.On {
		t.Error("want On=true")
	}
}

func TestParse_Sync_Off(t *testing.T) {
	cmd, err := Parse("sync off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := cmd.(Sync)
	if !ok || s.On {
		t.Errorf("want Sync{On:false}, got %+v", cmd)
	}
}

// ─── HUD ──────────────────────────────────────────────────────────────────────

func TestParse_HUD_On(t *testing.T) {
	cmd, err := Parse("hud on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h, ok := cmd.(HUD)
	if !ok || !h.Visible {
		t.Errorf("want HUD{Visible:true}, got %+v", cmd)
	}
}

func TestParse_HUD_Off(t *testing.T) {
	cmd, err := Parse("hud off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h, ok := cmd.(HUD)
	if !ok || h.Visible {
		t.Errorf("want HUD{Visible:false}, got %+v", cmd)
	}
}

func TestParse_HUD_List(t *testing.T) {
	cmd, err := Parse("hud list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(HUDList); !ok {
		t.Errorf("want HUDList, got %T", cmd)
	}
}

func TestParse_HUD_Category_On(t *testing.T) {
	cmd, err := Parse("hud debug on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hc, ok := cmd.(HUDCategory)
	if !ok {
		t.Fatalf("want HUDCategory, got %T", cmd)
	}
	if hc.Category != "debug" || !hc.Visible {
		t.Errorf("want {debug, true}, got %+v", hc)
	}
}

func TestParse_HUD_NoArg_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("hud")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

// ─── Record ───────────────────────────────────────────────────────────────────

func TestParse_Record_Start(t *testing.T) {
	cmd, err := Parse("record start tour.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rs, ok := cmd.(RecordStart)
	if !ok {
		t.Fatalf("want RecordStart, got %T", cmd)
	}
	if rs.Filename != "tour.mp4" {
		t.Errorf("want tour.mp4, got %q", rs.Filename)
	}
}

func TestParse_Record_Pause(t *testing.T) {
	cmd, err := Parse("record pause")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(RecordPause); !ok {
		t.Errorf("want RecordPause, got %T", cmd)
	}
}

func TestParse_Record_Stop(t *testing.T) {
	cmd, err := Parse("record stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(RecordStop); !ok {
		t.Errorf("want RecordStop, got %T", cmd)
	}
}

func TestParse_Record_Delete(t *testing.T) {
	cmd, err := Parse("record delete old.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rd, ok := cmd.(RecordDelete)
	if !ok {
		t.Fatalf("want RecordDelete, got %T", cmd)
	}
	if rd.Filename != "old.mp4" {
		t.Errorf("want old.mp4, got %q", rd.Filename)
	}
}

func TestParse_Record_NoSubcmd_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("record")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

// ─── Config ───────────────────────────────────────────────────────────────────

func TestParse_Config_ReloadKeybindings(t *testing.T) {
	cmd, err := Parse("config reload keybindings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(ConfigReloadKeybindings); !ok {
		t.Errorf("want ConfigReloadKeybindings, got %T", cmd)
	}
}

func TestParse_Config_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("config")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestParse_Config_Unknown_ReturnsError(t *testing.T) {
	_, err := Parse("config bogus arg")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Session ──────────────────────────────────────────────────────────────────

func TestParse_Session_Register(t *testing.T) {
	cmd, err := Parse("session register Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sr, ok := cmd.(SessionRegister)
	if !ok {
		t.Fatalf("want SessionRegister, got %T", cmd)
	}
	if sr.Label != "Alice" {
		t.Errorf("want Alice, got %q", sr.Label)
	}
}

func TestParse_Session_Unregister(t *testing.T) {
	cmd, err := Parse("session unregister")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(SessionUnregister); !ok {
		t.Errorf("want SessionUnregister, got %T", cmd)
	}
}

func TestParse_Session_List(t *testing.T) {
	cmd, err := Parse("session list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(SessionList); !ok {
		t.Errorf("want SessionList, got %T", cmd)
	}
}

func TestParse_Session_Kick(t *testing.T) {
	cmd, err := Parse("session kick abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sk, ok := cmd.(SessionKick)
	if !ok {
		t.Fatalf("want SessionKick, got %T", cmd)
	}
	if sk.TargetSessionID != "abc-123" {
		t.Errorf("want abc-123, got %q", sk.TargetSessionID)
	}
}

func TestParse_Session_Teleport(t *testing.T) {
	cmd, err := Parse("session teleport abc-123 Mars")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	st, ok := cmd.(SessionTeleport)
	if !ok {
		t.Fatalf("want SessionTeleport, got %T", cmd)
	}
	if st.TargetSessionID != "abc-123" || st.Body != "Mars" {
		t.Errorf("want {abc-123, Mars}, got %+v", st)
	}
}

func TestParse_Session_NoSubcmd_ReturnsError(t *testing.T) {
	_, err := Parse("session")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// ─── Help Keys ────────────────────────────────────────────────────────────────

func TestParse_Help_Keys(t *testing.T) {
	cmd, err := Parse("help keys")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(HelpKeys); !ok {
		t.Errorf("want HelpKeys, got %T", cmd)
	}
}

// ─── Shutdown ─────────────────────────────────────────────────────────────────

func TestParse_Shutdown(t *testing.T) {
	cmd, err := Parse("shutdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Shutdown); !ok {
		t.Errorf("want Shutdown, got %T", cmd)
	}
}

// ─── Clear ────────────────────────────────────────────────────────────────────

func TestParse_Clear(t *testing.T) {
	cmd, err := Parse("clear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Clear); !ok {
		t.Errorf("want Clear, got %T", cmd)
	}
}

// ─── Exit alias ───────────────────────────────────────────────────────────────

func TestParse_Exit_Alias(t *testing.T) {
	cmd, err := Parse("exit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Quit); !ok {
		t.Errorf("want Quit, got %T", cmd)
	}
}
