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
		input  string
		want   []string
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
