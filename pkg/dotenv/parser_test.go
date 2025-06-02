package dotenv

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "simple",
			input: "KEY=value",
			want:  map[string]string{"KEY": "value"},
		},
		{
			name: "multiple",
			input: `KEY1=value1
KEY2=value2
KEY3=value3`,
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
		},
		{
			name: "with_spaces",
			input: `KEY1 = value1
KEY2=value with spaces
KEY3=  leading and trailing  `,
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value with spaces",
				"KEY3": "leading and trailing",
			},
		},
		{
			name: "quoted_values",
			input: `DOUBLE="double quoted"
SINGLE='single quoted'
MIXED="double with 'single' inside"
ESCAPE="escaped \"quote\""`,
			want: map[string]string{
				"DOUBLE": "double quoted",
				"SINGLE": "single quoted",
				"MIXED":  "double with 'single' inside",
				"ESCAPE": `escaped "quote"`,
			},
		},
		{
			name: "escape_sequences",
			input: `NEWLINE="line1\nline2"
TAB="col1\tcol2"
RETURN="text\r"
BACKSLASH="path\\to\\file"
HEX="prefix\x41suffix"`,
			want: map[string]string{
				"NEWLINE":   "line1\nline2",
				"TAB":       "col1\tcol2",
				"RETURN":    "text\r",
				"BACKSLASH": "path\\to\\file",
				"HEX":       "prefixAsuffix",
			},
		},
		{
			name: "comments_and_empty",
			input: `# This is a comment
KEY1=value1

# Another comment
KEY2=value2
`,
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "export_format",
			input: `export KEY1=value1
export KEY2="value2"
export KEY3='value3'`,
			want: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
		},
		{
			name:  "empty_value",
			input: "KEY=",
			want:  map[string]string{"KEY": ""},
		},
		{
			name: "unicode",
			input: `EMOJI=🚀
CHINESE=你好
MIXED="Hello 世界"`,
			want: map[string]string{
				"EMOJI":   "🚀",
				"CHINESE": "你好",
				"MIXED":   "Hello 世界",
			},
		},
		{
			name:    "invalid_no_equals",
			input:   "INVALID",
			wantErr: true,
		},
		{
			name:    "invalid_key_start_number",
			input:   "123KEY=value",
			wantErr: true,
		},
		{
			name:    "invalid_key_special_char",
			input:   "KEY-NAME=value",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			reader := strings.NewReader(tt.input)
			
			got, err := p.Parse(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ExpandVars(t *testing.T) {
	// Set up test environment variable
	os.Setenv("TEST_ENV_VAR", "from_env")
	defer os.Unsetenv("TEST_ENV_VAR")
	
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "expand_existing",
			input: `BASE=base_value
EXPANDED=$BASE/suffix
BRACES=${BASE}/suffix`,
			want: map[string]string{
				"BASE":     "base_value",
				"EXPANDED": "base_value/suffix",
				"BRACES":   "base_value/suffix",
			},
		},
		{
			name: "expand_env",
			input: `FROM_ENV=$TEST_ENV_VAR
COMBINED=${TEST_ENV_VAR}_suffix`,
			want: map[string]string{
				"FROM_ENV": "from_env",
				"COMBINED": "from_env_suffix",
			},
		},
		{
			name: "expand_not_found",
			input: `MISSING=$NONEXISTENT
MISSING_BRACES=${NONEXISTENT}`,
			want: map[string]string{
				"MISSING":        "$NONEXISTENT",
				"MISSING_BRACES": "${NONEXISTENT}",
			},
		},
		{
			name: "no_expand_single_quotes",
			input: `LITERAL='$BASE'
DOUBLE="$BASE"`,
			want: map[string]string{
				"LITERAL": "$BASE",
				"DOUBLE":  "$BASE", // Not expanded because BASE not defined
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.ExpandVars = true
			
			reader := strings.NewReader(tt.input)
			got, err := p.Parse(reader)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_Configuration(t *testing.T) {
	input := `# Comment
  KEY1 = value1  
KEY2=value2
INVALID LINE
KEY3=value3`
	
	tests := []struct {
		name      string
		configure func(*Parser)
		wantKeys  []string
		wantErr   bool
	}{
		{
			name: "default",
			configure: func(p *Parser) {
				// Use defaults
			},
			wantKeys: []string{"KEY1", "KEY2"},
			wantErr:  true, // Invalid line causes error
		},
		{
			name: "ignore_invalid",
			configure: func(p *Parser) {
				p.IgnoreInvalid = true
			},
			wantKeys: []string{"KEY1", "KEY2", "KEY3"},
			wantErr:  false,
		},
		{
			name: "no_trim",
			configure: func(p *Parser) {
				p.TrimSpace = false
			},
			wantKeys: []string{"KEY2"}, // KEY1 has spaces
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			tt.configure(p)
			
			reader := strings.NewReader(input)
			got, err := p.Parse(reader)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err == nil {
				if len(got) != len(tt.wantKeys) {
					t.Errorf("Parse() returned %d keys, want %d", len(got), len(tt.wantKeys))
				}
				
				for _, key := range tt.wantKeys {
					if _, ok := got[key]; !ok {
						t.Errorf("Missing expected key: %s", key)
					}
				}
			}
		})
	}
}

func TestParser_ParseFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	
	content := `KEY1=value1
KEY2=value2
# Comment
KEY3=value3`
	
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	p := NewParser()
	got, err := p.ParseFile(envFile)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	
	want := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}
	
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseFile() = %v, want %v", got, want)
	}
	
	// Test non-existent file
	_, err = p.ParseFile(filepath.Join(tmpDir, "nonexistent.env"))
	if err == nil {
		t.Error("ParseFile() expected error for non-existent file")
	}
}

func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "KEY", true},
		{"underscore_start", "_KEY", true},
		{"with_number", "KEY123", true},
		{"all_caps", "MY_VAR_NAME", true},
		{"lowercase", "myvar", true},
		{"mixed", "MyVar_123", true},
		
		{"empty", "", false},
		{"start_number", "123KEY", false},
		{"with_dash", "KEY-NAME", false},
		{"with_space", "KEY NAME", false},
		{"with_dot", "KEY.NAME", false},
		{"with_dollar", "KEY$", false},
		{"special_chars", "KEY@#", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidEnvVarName(tt.input); got != tt.want {
				t.Errorf("isValidEnvVarName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParser_ParseWithMetadata(t *testing.T) {
	input := `KEY1=value1
# Comment line
KEY2=value2

KEY3=value3`
	
	p := NewParser()
	reader := strings.NewReader(input)
	
	vars, err := p.ParseWithMetadata(reader)
	if err != nil {
		t.Fatalf("ParseWithMetadata() error = %v", err)
	}
	
	if len(vars) != 3 {
		t.Errorf("ParseWithMetadata() returned %d variables, want 3", len(vars))
	}
	
	// Check metadata
	expectedLines := []int{1, 3, 5}
	for i, v := range vars {
		if v.LineNum != expectedLines[i] {
			t.Errorf("Variable %d LineNum = %d, want %d", i, v.LineNum, expectedLines[i])
		}
		
		if v.Original == "" {
			t.Errorf("Variable %d missing Original line", i)
		}
	}
}

func TestParser_ParseWithStats(t *testing.T) {
	input := `# Header comment
KEY1=value1
KEY2=value2

# Another comment
KEY3=value3
KEY1=duplicate
INVALID LINE
KEY4=value4
`
	
	p := NewParser()
	p.IgnoreInvalid = true
	reader := strings.NewReader(input)
	
	vars, stats, err := p.ParseWithStats(reader)
	if err != nil {
		t.Fatalf("ParseWithStats() error = %v", err)
	}
	
	// Check variables
	if len(vars) != 4 {
		t.Errorf("ParseWithStats() returned %d variables, want 4", len(vars))
	}
	
	// Check stats
	if stats.TotalLines != 10 { // Including empty line at end
		t.Errorf("TotalLines = %d, want 10", stats.TotalLines)
	}
	if stats.Variables != 5 { // Including duplicate
		t.Errorf("Variables = %d, want 5", stats.Variables)
	}
	if stats.Comments != 2 {
		t.Errorf("Comments = %d, want 2", stats.Comments)
	}
	if stats.EmptyLines != 2 {
		t.Errorf("EmptyLines = %d, want 2", stats.EmptyLines)
	}
	if stats.InvalidLines != 1 {
		t.Errorf("InvalidLines = %d, want 1", stats.InvalidLines)
	}
	if len(stats.DuplicateKeys) != 1 || stats.DuplicateKeys[0] != "KEY1" {
		t.Errorf("DuplicateKeys = %v, want [KEY1]", stats.DuplicateKeys)
	}
}

func TestParser_ComplexQuoting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "mixed_quotes",
			input: `KEY="She said 'hello'"`,
			want:  "She said 'hello'",
		},
		{
			name:  "escaped_in_double",
			input: `KEY="Line1\nLine2\tTabbed"`,
			want:  "Line1\nLine2\tTabbed",
		},
		{
			name:  "no_escape_in_single",
			input: `KEY='Line1\nLine2'`,
			want:  `Line1\nLine2`,
		},
		{
			name:  "unmatched_quote",
			input: `KEY="unclosed`,
			want:  `"unclosed`,
		},
		{
			name:  "empty_quotes",
			input: `KEY=""`,
			want:  "",
		},
		{
			name:  "quotes_in_unquoted",
			input: `KEY=value"with"quotes`,
			want:  `value"with"quotes`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			vars, err := p.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			
			if got := vars["KEY"]; got != tt.want {
				t.Errorf("Parse() value = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "equals_in_value",
			input: `URL=postgres://user:pass@host:5432/db?ssl=true`,
			want:  map[string]string{"URL": "postgres://user:pass@host:5432/db?ssl=true"},
		},
		{
			name:  "multiple_equals",
			input: `EQUATION=a=b+c=d`,
			want:  map[string]string{"EQUATION": "a=b+c=d"},
		},
		{
			name:  "trailing_backslash",
			input: `PATH=C:\Users\`,
			want:  map[string]string{"PATH": `C:\Users\`},
		},
		{
			name:  "just_key_equals",
			input: `KEY=`,
			want:  map[string]string{"KEY": ""},
		},
		{
			name:    "just_equals",
			input:   `=value`,
			wantErr: true,
		},
		{
			name:  "spaces_around_equals",
			input: `KEY = value = with = equals`,
			want:  map[string]string{"KEY": "value = with = equals"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.Parse(strings.NewReader(tt.input))
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParser_Parse(b *testing.B) {
	// Create a realistic .env file content
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "KEY_%d=value_%d\n", i, i)
		if i%10 == 0 {
			fmt.Fprintln(&buf, "# Comment line")
		}
		if i%20 == 0 {
			fmt.Fprintln(&buf) // Empty line
		}
	}
	
	content := buf.String()
	p := NewParser()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, err := p.Parse(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_ExpandVars(b *testing.B) {
	content := `BASE=/usr/local
BIN=$BASE/bin
LIB=${BASE}/lib
PATH=$BIN:$LIB:$PATH`
	
	p := NewParser()
	p.ExpandVars = true
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, err := p.Parse(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}