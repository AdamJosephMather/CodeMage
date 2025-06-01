package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"golang.design/x/clipboard"
)

type Line struct {
	text string
	changed bool
	styles []tcell.Style
	start_str bool
	start_str_type rune
	end_str bool
	end_str_type rune
}

type Edit struct {
	row int
	col int
	width int
	height int
	
	buffer []Line
	
	toprow int
	leftchar int
	cursor Cursor
	use_line_numbers bool
	
	current_mode string
	number_string string
	
	UNDO_HISTORY []Snapshot
	REDO_HISTORY []Snapshot
}

type Cursor struct {
	row int
	col int
	row_anchor int
	col_anchor int
	preferencial_col int
}

type Snapshot struct {
	buffer []Line
	cursor Cursor
	time_taken int64
	fulltext string
}

var version string = "0.0.1"

var s tcell.Screen
var current_window string
var file_name string
var absolute_path string
var opening_file string
var title string

var DEF_STYLE tcell.Style
var INVERTED_STYLE tcell.Style
var TITLE_STYLE tcell.Style
var HIGHLIGHT_STYLE tcell.Style
var LINE_NUMBER_STYLE tcell.Style
var STRING_STYLE tcell.Style
var NORMAL_MODE_STYLE tcell.Style
var FUNCTION_STYLE tcell.Style
var KEYWORD_STYLE tcell.Style
var NAME_STYLE tcell.Style
var PUNC_STYLE tcell.Style
var COMMENT_STYLE tcell.Style
var LITTERAL_STYLE tcell.Style
var SPECIAL_STYLE tcell.Style

var KEYWORDS []string = []string{"if", "elif", "else", "var", "let", "const", "mut", "return", "break", "yield", "continue", "case", "switch", "func", "def", "fun", "function", "define", "import", "for", "while", "type", "struct", "package", "nil", "false", "true", "none", "False", "True", "None", "Null", "null", "try", "catch", "except", "default"}

var MAIN_TEXTEDIT Edit
var INPT_TEXTEDIT Edit
var FIND_TEXTEDIT Edit
var REPLACE_TEXTEDIT Edit
var SHOWING_FIND bool
var USING_REPLACE bool // if false, find, else replace
var INPUT_MODAL_CALLBACK func() = nil
var SHOWING_INPUT_MODAL bool
var SHOWING_INPUT_BOOL bool
var CURRENT_SELECTED_BOOL bool
var INPUT_MODAL_LABEL string
var BUTTON_DOWN bool

var MOVE_DOWN = 1
var MOVE_UP = 2
var MOVE_LEFT = 3
var MOVE_RIGHT = 4
var WORD_LEFT = 7
var WORD_RIGHT = 8

var BACKSPACE = 5
var DELETE = 6

var WHITESPACE = " \t"
var PUNCTUATION = "./>,<-=+[]{}|\\)(*&^%$#@!`~:;'\"?"
var NUMBERS = "0123456789"

var NORMAL_CHAR_TYPE = 9
var WHITESPACE_CHAR_TYPE = 10
var PUNCTUATION_CHAR_TYPE = 11

var BACKSPACE_WORD = 12
var DELETE_WORD = 13
var END_OF_LINE = 14
var START_OF_LINE = 15
var FULL_END = 16

var LAST_SAVED string
var NEED_TO_EXIT bool
var SAVE_CALLBACK func() = nil
var CHECK_FOR_SAVE_CALLBACK func() = nil

var APP_CONFIG_DIR string

var titleColor = tcell.NewRGBColor(25, 25, 25)
var highlightColor = tcell.NewRGBColor(100, 100, 100)
var lineNumberColor = tcell.NewRGBColor(50, 50, 50)
var colorSTRING = tcell.NewRGBColor(127, 173, 94)
var colorFUNCTION = tcell.NewRGBColor(199, 157, 78)
var colorKEYWORD = tcell.NewRGBColor(176, 95, 199)
var colorNAME = tcell.NewRGBColor(245, 91, 102)
var colorPUNC = tcell.NewRGBColor(127, 132, 142)
var colorCOMMENT = tcell.NewRGBColor(127, 132, 142)
var colorLITTERAL = tcell.NewRGBColor(194, 127, 64)
var colorBackground = tcell.NewRGBColor(15, 15, 15)
var colorSpecial = tcell.NewRGBColor(219, 150, 53)

var CURRENT_TEXT_EDIT string = "main"

var SCROLL_SENSITIVITY int

func emitStr(x, y int, style tcell.Style, str string) {
	for i, r := range []rune(str) {
		s.SetContent(x+i, y, r, nil, style)
	}
}

func emitStrColored(x, y int, style []tcell.Style, str string) {
	for i, r := range []rune(str) {
		s.SetContent(x+i, y, r, nil, style[i])
	}
}

func createEdit() Edit {
	width, height := s.Size()
	
	buffer := make([]Line, 1)
	old_buffer := make([]string, 1)
	
	cursor := Cursor{row: 0, col: 0, row_anchor: 0, col_anchor: 0}
	buffer[0] = Line{text: "", end_str: false}
	old_buffer[0] = ""
	
	edit := Edit{row: 1, col: 0, width: width, height: height-1, buffer: buffer, cursor: cursor, toprow: 0, leftchar: 0, use_line_numbers: true, current_mode: "i", number_string: ""}
	
	readyUndoHistory(&edit)
	
	edit.UNDO_HISTORY[0].time_taken = 0 // ensuring it stays as independant event
	
	return edit
}

func setupUI() {
	width, height := s.Size()
	
	MAIN_TEXTEDIT = createEdit()
	
	INPT_TEXTEDIT = createEdit()
	INPT_TEXTEDIT.width = 30
	INPT_TEXTEDIT.height = 3
	INPT_TEXTEDIT.row = height/2-1
	INPT_TEXTEDIT.col = (width-INPT_TEXTEDIT.width)/2
	INPT_TEXTEDIT.use_line_numbers = false
	
	FIND_TEXTEDIT = createEdit()
	FIND_TEXTEDIT.height = 1
	FIND_TEXTEDIT.width = width-4
	FIND_TEXTEDIT.row = height-4
	FIND_TEXTEDIT.col = 2
	FIND_TEXTEDIT.use_line_numbers = false
	
	REPLACE_TEXTEDIT = createEdit()
	REPLACE_TEXTEDIT.height = 1
	REPLACE_TEXTEDIT.width = width-4
	REPLACE_TEXTEDIT.row = height-2
	REPLACE_TEXTEDIT.col = 2
	REPLACE_TEXTEDIT.use_line_numbers = false
	
	SHOWING_INPUT_MODAL = false
	SHOWING_INPUT_BOOL = false
	CURRENT_SELECTED_BOOL = true
	CURRENT_TEXT_EDIT = "main"
	
	redrawFullScreen()
}

func createNew() {
	title = "Untitled"
	file_name = ""
	setupUI()
}

func checkForStyleUpdates(edit *Edit) {
	for indx, line := range(edit.buffer) {
		var preline Line
		
		if indx > 0 {
			preline = edit.buffer[indx-1]
		}else{
			preline = Line{}
		}
		
		if !line.changed && line.start_str == preline.end_str && line.start_str_type == preline.end_str_type {
			continue
		}
		
		line.styles = []tcell.Style{}
		line.changed = false
		line.start_str = preline.end_str
		line.start_str_type = preline.end_str_type
		
		cur_str := line.start_str
		cur_str_type := line.start_str_type
		is_real := true
		in_comment := false
		prechar := ' '
		was_name := false
		start_of_name := 0
		name := ""
		
		was_literal := false
		
		for indx_c, char := range(line.text) {
			is_name := false
			is_literal := false
			
			if in_comment {
				line.styles = append(line.styles, COMMENT_STYLE)
			}else if cur_str {
				line.styles = append(line.styles, STRING_STYLE)
				
				if char == cur_str_type && is_real {
					cur_str = false
				}
				
				if char == '\\'{
					is_real = !is_real
				}else{
					is_real = true
				}
				
			}else if char == '"' || char == '\'' {
				cur_str = true
				is_real = true
				line.styles = append(line.styles, STRING_STYLE)
				cur_str_type = char
			}else if strings.Contains(PUNCTUATION, string(char)) && !(char == '.' && was_literal) {
				if char == '#' { // python comment
					in_comment = true
					line.styles = append(line.styles, COMMENT_STYLE)
				}else if char == '/' && prechar == '/'{ // any other lang - this will f*** w/ python // sign...
					in_comment = true
					line.styles[len(line.styles)-1] = COMMENT_STYLE // retroactively change the last char
					line.styles = append(line.styles, COMMENT_STYLE)
				}else{
					line.styles = append(line.styles, PUNC_STYLE)
				}
				is_real = true
			}else if strings.Contains(WHITESPACE, string(char)) {
				line.styles = append(line.styles, DEF_STYLE)
				
				is_real = true
			}else if (strings.Contains(NUMBERS, string(char)) && !was_name) || char == '.' && was_literal {
				if !was_literal {
					was_literal = true
				}
				is_literal = true
				line.styles = append(line.styles, LITTERAL_STYLE)
				is_real = true
			}else { // part of id (name)
				line.styles = append(line.styles, NAME_STYLE)
				is_real = true
				
				if !was_name{
					start_of_name = indx_c
					was_name = true
					name = ""
				}
				
				name += string(char)
				
				is_name = true
			}
			
			if !is_name && was_name {
				if char == '(' {
					for rep := range(len(line.styles)-start_of_name-1) {
						line.styles[rep+start_of_name] = FUNCTION_STYLE
					}
				}else if slices.Contains(KEYWORDS, name) {
					for rep := range(len(line.styles)-start_of_name-1) {
						line.styles[rep+start_of_name] = KEYWORD_STYLE
					}
				}
				was_name = false
			}
			
			was_literal = is_literal
			
			prechar = char
		}
		
		if was_name {
			if slices.Contains(KEYWORDS, name) {
				for rep := range(len(line.styles)-start_of_name) {
					line.styles[rep+start_of_name] = KEYWORD_STYLE
				}
			}
		}
		
		line.end_str = cur_str
		line.end_str_type = cur_str_type
		
		edit.buffer[indx] = line
	}
}

func repeatSlice[T any](s T, n int) []T {
	if n <= 0 {
		return []T{} // Return an empty slice if n is zero or negative
	}

	repeated := make([]T, n) // Pre-allocate capacity for efficiency
	
	for i := range(n) {
		repeated[i] = s
	}
	
	return repeated
}

func drawEdit(edit *Edit, is_current bool) {
	checkForStyleUpdates(edit)
	
	buffer := edit.buffer
	cursor := edit.cursor
	
	line_num_width := len(strconv.Itoa(len(buffer)))
	
	if !edit.use_line_numbers {
		line_num_width = 0
	}
	
	cursor_pos := cursor.row
	
	for yraw := range(edit.height) {
		y := yraw + edit.row
		
		line_num := edit.toprow+yraw // 0 based
		
		if line_num >= len(buffer) && edit.use_line_numbers{
			emitStr(edit.col, y, LINE_NUMBER_STYLE, strings.Repeat(" ", line_num_width-1)+"~"+strings.Repeat(" ", edit.width-line_num_width))
			emitStr(edit.col+line_num_width, y, DEF_STYLE, strings.Repeat(" ", edit.width-line_num_width))
			continue
		}else if line_num >= len(buffer){
			emitStr(edit.col, y, DEF_STYLE, strings.Repeat(" ", edit.width))
			continue
		}
		
		if edit.use_line_numbers {
			rel_line_num := cursor_pos-line_num
			on_end := false
			
			if rel_line_num < 0 {
				rel_line_num = -rel_line_num
			}else if rel_line_num == 0{
				rel_line_num = line_num+1 // convert out of zero based
				on_end = true
			}
			
			line_rel_str := strconv.Itoa(rel_line_num)
			num_spaces := line_num_width - len(line_rel_str)
			
			fullstr := strings.Repeat(" ", num_spaces)+line_rel_str
			if on_end {
				fullstr = line_rel_str+strings.Repeat(" ", num_spaces)
			}
			
			emitStr(edit.col, y, LINE_NUMBER_STYLE, fullstr)
		}
		
		
		// detect if it's in the cursor selection range
		
		minRng := -1
		maxRng := -1
		
		if cursor.col != cursor.col_anchor || cursor.row != cursor.row_anchor {
			// it is a worry here (cursor is selecting something)
			end_row := cursor.row
			end_col := cursor.col
			start_row := cursor.row_anchor
			start_col := cursor.col_anchor
				
			if end_row < start_row {
				start_row, end_row = end_row, start_row
				start_col, end_col = end_col, start_col
			}else if end_row == start_row && start_col > end_col {
				end_col, start_col = start_col, end_col
			}
			
			if line_num > start_row && line_num < end_row {
				minRng = -1
				maxRng = len(edit.buffer[line_num].text)+1
			}else if line_num == start_row && line_num == end_row {
				minRng = start_col-1
				maxRng = end_col
			}else if line_num == start_row {
				minRng = start_col-1
				maxRng = len(edit.buffer[line_num].text)+1
			}else if line_num == end_row {
				minRng = -1
				maxRng = end_col
			}
		}
				
		lineToDraw := ""
		tru_col_current := 0
		
		runes := []rune(buffer[line_num].text)
		exist_styles := buffer[line_num].styles
		exist_styles_len := len(exist_styles)
		
		charIndx := 0
		
		curs_line := cursor_pos == line_num
		curs_char := cursor.col
		
		styles := []tcell.Style{}
		
		for true {
			is_cursor := curs_line && charIndx == curs_char && is_current
			is_in_highlight := charIndx > minRng && charIndx < maxRng
			
			cur_style := DEF_STYLE
			if charIndx < exist_styles_len {
				cur_style = exist_styles[charIndx]
			}
			
			if is_cursor && edit.current_mode == "i" {
				cur_style = INVERTED_STYLE
			}else if is_cursor && edit.current_mode == "n" {
				cur_style = NORMAL_MODE_STYLE
			}else if is_in_highlight {
				cur_style = HIGHLIGHT_STYLE
			}
			
			if charIndx >= len(runes) {
				lineToDraw += strings.Repeat(" ", edit.width-len(lineToDraw))
				styles = append(styles, cur_style)
				styles = append(styles, repeatSlice(DEF_STYLE, edit.width-len(styles))...)
				break
			}
			
			char := runes[charIndx]
			
			if char != '\t'{
				if tru_col_current >= edit.leftchar {
					lineToDraw += string(char)
					styles = append(styles, cur_style)
				}
				tru_col_current ++
			}else{
				for tab_indx := range(4) {
					if tru_col_current >= edit.leftchar {
						lineToDraw += " "
						if tab_indx == 0 {
							styles = append(styles, cur_style)
						}else if is_in_highlight {
							styles = append(styles, HIGHLIGHT_STYLE)
						}else{
							styles = append(styles, DEF_STYLE)
						}
					}
					tru_col_current ++
					
					if len(lineToDraw) == edit.width {break}
				}
			}
			
			charIndx ++
			
			if len(lineToDraw) == edit.width {
				break
			}
		}
		
		x := edit.col+line_num_width
		emitStrColored(x, y, styles, lineToDraw)
	}
}

func drawYesNo() {
	s1 := INVERTED_STYLE
	s2 := DEF_STYLE
	
	if !CURRENT_SELECTED_BOOL {
		s1, s2 = s2, s1
	}
	
	for line := range INPT_TEXTEDIT.height {
		emitStr(INPT_TEXTEDIT.col, INPT_TEXTEDIT.row+line, DEF_STYLE, strings.Repeat(" ", INPT_TEXTEDIT.width))
	}
	
	emitStr(INPT_TEXTEDIT.col+2, INPT_TEXTEDIT.row+1, s1, "Yes")
	emitStr(INPT_TEXTEDIT.col+INPT_TEXTEDIT.width-4, INPT_TEXTEDIT.row+1, s2, "No")
}

func drawOutline(edit *Edit, style tcell.Style, text string) {
	emitStr(edit.col-2, edit.row-1, style, text+strings.Repeat(" ", edit.width+4-len(text)))
	emitStr(edit.col-2, edit.row+edit.height, style, strings.Repeat(" ", edit.width+4))
	
	for row := range edit.height {
		emitStr(edit.col-2, edit.row+row, style, "  ")
		emitStr(edit.col+edit.width, edit.row+row, style, "  ")
	}
}

func drawFullEdit() {
	drawEdit(&MAIN_TEXTEDIT, CURRENT_TEXT_EDIT == "main")
	
	if SHOWING_INPUT_MODAL {
		drawOutline(&INPT_TEXTEDIT, TITLE_STYLE, INPUT_MODAL_LABEL)
		drawEdit(&INPT_TEXTEDIT, CURRENT_TEXT_EDIT == "inpt")
	}else if SHOWING_INPUT_BOOL {
		drawOutline(&INPT_TEXTEDIT, TITLE_STYLE, INPUT_MODAL_LABEL)
		drawYesNo()
	}
	
	if SHOWING_FIND {
		drawEdit(&FIND_TEXTEDIT, CURRENT_TEXT_EDIT == "find")
		drawEdit(&REPLACE_TEXTEDIT, CURRENT_TEXT_EDIT == "replace")
		drawOutline(&FIND_TEXTEDIT, TITLE_STYLE, "Find Text")
		drawOutline(&REPLACE_TEXTEDIT, TITLE_STYLE, "Replace With")
	}
	
	drawTitleBar()
}

func drawTitleBar() {
	w, _ := s.Size()
	
	text := "CodeMage V"+version+" - "+title
	text += strings.Repeat(" ", w-len(text))
	emitStr(0, 0, TITLE_STYLE, text)
	
	text = "ERROR IN MAKING THE TITLEBAR?"
	if MAIN_TEXTEDIT.current_mode == "n" {
		text = "NORMAL"
	}else if MAIN_TEXTEDIT.current_mode == "i" {
		text = "INSERT"
	}
	emitStr(w-len(text), 0, TITLE_STYLE, text)
}

func redrawFullScreen() {
	s.Sync()
	s.Clear()
	width, height := s.Size()
	
	if current_window == "blank"{
		lines := []string{"CodeMage V"+version, "Designed by Adam Mather"}
		
		for i := range(lines){
			line := lines[i]
			linelen := len(line)
			startX := (width-linelen)/2
			startY := (height/2)+i-1
			
			emitStr(startX, startY, SPECIAL_STYLE, line)
		}
	}else if current_window == "edit" {
		MAIN_TEXTEDIT.width = width
		
		if SHOWING_FIND {
			MAIN_TEXTEDIT.height = height-6
			
			if MAIN_TEXTEDIT.height < 1 {
				MAIN_TEXTEDIT.height = 1
			}
		}else{
			MAIN_TEXTEDIT.height = height-1
		}
		
		INPT_TEXTEDIT.width = 30
		INPT_TEXTEDIT.height = 3
		INPT_TEXTEDIT.row = height/2-1
		INPT_TEXTEDIT.col = (width-INPT_TEXTEDIT.width)/2
		
		FIND_TEXTEDIT.width = width-4
		FIND_TEXTEDIT.height = 1
		FIND_TEXTEDIT.row = height-4
		FIND_TEXTEDIT.col = 2
		
		REPLACE_TEXTEDIT.width = width-4
		REPLACE_TEXTEDIT.height = 1
		REPLACE_TEXTEDIT.row = height-2
		REPLACE_TEXTEDIT.col = 2
		
		drawFullEdit()
	}
	
	s.Show()
}

func ContainsString(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true // Found the string
		}
	}
	return false // String not found in the slice
}

func deleteText(mode, repeat int, edit *Edit) {
	if edit.cursor.row != edit.cursor.row_anchor || edit.cursor.col != edit.cursor.col_anchor {
		// find the bottom of the selection, calculate the number of characters to delete, call delete with said repeat, exit.
		cursor := edit.cursor
		
		end_row := cursor.row
		end_col := cursor.col
		start_row := cursor.row_anchor
		start_col := cursor.col_anchor
			
		if end_row < start_row {
			start_row, end_row = end_row, start_row
			start_col, end_col = end_col, start_col
		}else if end_row == start_row && end_col < start_col {
			start_col, end_col = end_col, start_col
		}
		
		if start_row == end_row {
			line := edit.buffer[start_row]
			
			start := line.text[:start_col]
			end := line.text[end_col:]
			
			edit.buffer[start_row].text = start + end // delete section
			edit.buffer[start_row].changed = true // delete section
		}else{
			first_line := edit.buffer[start_row].text[:start_col]
			last_line := edit.buffer[end_row].text[end_col:]
			edit.buffer[start_row].text = first_line + last_line
			edit.buffer[start_row].changed = true
			
			edit.buffer = append(edit.buffer[:start_row+1], edit.buffer[end_row+1:]...)
		}
		
		edit.cursor.row = start_row
		edit.cursor.col = start_col
		edit.cursor.row_anchor = start_row
		edit.cursor.col_anchor = start_col
		edit.cursor.preferencial_col = getTrueCol(start_col, start_row, edit)
		
		return
	}
	
	for range(repeat){
		if mode == BACKSPACE {
			line := edit.buffer[edit.cursor.row].text
			after_deleted := ""
			
			if edit.cursor.col < len(line){
				after_deleted = line[edit.cursor.col:]
			}
			before_deleted := ""
			if edit.cursor.col > 0{
				before_deleted = line[:edit.cursor.col-1]
			}
			
			if edit.cursor.col == 0 && edit.cursor.row != 0 {
				edit.cursor.row--
				
				joining_line := edit.buffer[edit.cursor.row].text
				
				edit.cursor.col = len(joining_line)
				
				
				edit.buffer[edit.cursor.row].text += after_deleted
				edit.buffer[edit.cursor.row].changed = true
				edit.buffer = append(edit.buffer[:edit.cursor.row+1], edit.buffer[edit.cursor.row+2:]...)
			}else{
				moveCursor(MOVE_LEFT, false, 1, edit)
				edit.buffer[edit.cursor.row].text = before_deleted+after_deleted
				edit.buffer[edit.cursor.row].changed = true
			}
			
			edit.cursor.row_anchor = edit.cursor.row
			edit.cursor.col_anchor = edit.cursor.col
			edit.cursor.preferencial_col = getTrueCol(edit.cursor.col, edit.cursor.row, edit)
		}else if mode == DELETE {
			moveCursor(MOVE_RIGHT, false, 1, edit)
			deleteText(BACKSPACE, 1, edit)
		}else if mode == BACKSPACE_WORD {
			edit.cursor.col_anchor, edit.cursor.row_anchor = movePointInText(edit.cursor.col_anchor, edit.cursor.row_anchor, WORD_LEFT, 1, edit)
			deleteText(BACKSPACE, 1, edit)
		}else if mode == DELETE_WORD {
			edit.cursor.col_anchor, edit.cursor.row_anchor = movePointInText(edit.cursor.col_anchor, edit.cursor.row_anchor, WORD_RIGHT, 1, edit)
			deleteText(BACKSPACE, 1, edit)
		}
	}
}

func insertText(edit *Edit, text string) {
	if edit.cursor.row != edit.cursor.row_anchor || edit.cursor.col != edit.cursor.col_anchor {
		deleteText(BACKSPACE, 1, edit) // clear selection
	}
	
	lines_strings := strings.Split(text, "\n")
	
	lines := []Line{}
	for _, str := range(lines_strings) {
		lines = append(lines, Line{text: strings.ReplaceAll(str, "\r", ""), changed: true})
	}
	
	first_len := len(lines[0].text)
	end_len := len(lines[len(lines)-1].text)
	
	current_line := edit.buffer[edit.cursor.row].text
	bfrCrs := current_line[:edit.cursor.col]
	endCrs := current_line[edit.cursor.col:]
	
	lines[0].text = bfrCrs + lines[0].text
	lines[len(lines)-1].text = lines[len(lines)-1].text + endCrs
	
	edit.buffer[edit.cursor.row].text = lines[0].text
	edit.buffer[edit.cursor.row].changed = true
	
	end_line := edit.cursor.row
	end_char := edit.cursor.col + first_len
	
	buffer := edit.buffer
	
	if len(lines) > 1 {
		insertionIndex := edit.cursor.row+1
		
		start := append([]Line(nil), buffer[:insertionIndex]...) // ensure copy not reference
		end := buffer[insertionIndex:]
		
		newSlice := append(start, lines[1:]...)
		newSlice = append(newSlice, end...)
		
		edit.buffer = newSlice
		
		end_line = edit.cursor.row + len(lines)-1
		end_char = end_len
	}
	
	edit.cursor.row = end_line
	edit.cursor.col = end_char
	edit.cursor.row_anchor = end_line
	edit.cursor.col_anchor = end_char
	edit.cursor.preferencial_col = getTrueCol(end_char, end_line, edit)
}

func getCursorSelection(edit *Edit) string {
	s_c, s_r := edit.cursor.col, edit.cursor.row
	e_c, e_r := edit.cursor.col_anchor, edit.cursor.row_anchor
	
	if s_c == e_c && s_r == e_r {
		return ""
	}
	
	if e_r < s_r {
		s_r, e_r = e_r, s_r
		s_c, e_c = e_c, s_c
	}else if e_r == s_r && s_c > e_c {
		e_c, s_c = s_c, e_c
	}
	
	if s_r == e_r {
		return edit.buffer[s_r].text[s_c:e_c]
	}else{
		lines := append([]Line(nil), edit.buffer[s_r:e_r+1]...)
		lines[0].text = lines[0].text[s_c:]
		lines[len(lines)-1].text = lines[len(lines)-1].text[:e_c]
		
		line_str := []string{}
		for _, ln := range(lines) {
			line_str = append(line_str, ln.text)
		}
		return strings.Join(line_str, "\n")
	}
}

func insertNewLine(edit *Edit) {
	curLine := edit.buffer[edit.cursor.row].text
	
	tabs := ""
	for _, char := range(curLine){
		if char == '\t' {
			tabs += "\t"
		}else {
			break
		}
	}
	
	if len(curLine) != 0{
		lastchar := curLine[len(curLine)-1]
		if lastchar == ':' || lastchar == '(' || lastchar == '[' || lastchar == '{' {
			tabs += "\t"
		}
	}
	
	insertText(edit, "\n"+tabs)
}

func boolHandleKey(ev *tcell.EventKey) {
	rawrune := ev.Rune()
	
	rune := unicode.ToLower(rawrune)
	
	if rune == 'y' {
		CURRENT_SELECTED_BOOL = true
	}else if rune == 'n' {
		CURRENT_SELECTED_BOOL = false
	}else if ev.Key() == tcell.KeyEnter {
		SHOWING_INPUT_BOOL = false
		if INPUT_MODAL_CALLBACK != nil {
			INPUT_MODAL_CALLBACK()
		}
	}
	
	if rune == 'h' || rune == 'l' || ev.Key() == tcell.KeyLeft || ev.Key() == tcell.KeyRight || ev.Key() == tcell.KeyTab {
		CURRENT_SELECTED_BOOL = !CURRENT_SELECTED_BOOL
	}
}

func editHandleKey(ev *tcell.EventKey, edit *Edit) bool {
	rawrune := ev.Rune()
	
	rune := unicode.ToLower(rawrune)
	
	control_held := ev.Modifiers()&tcell.ModCtrl  != 0
	alt_held     := ev.Modifiers()&tcell.ModAlt   != 0
	shift_held   := ev.Modifiers()&tcell.ModShift != 0
	keepAnchor   := ev.Modifiers()&tcell.ModShift != 0
	handled := false
	
	if ev.Key() == tcell.KeyCtrlQ {
		return true
	}
	
	if ev.Key() == tcell.KeyCtrlY {
		redo(edit)
		showCursor(edit)
		return false // only can return here because it is not going to change the undo/redo history
	}else if ev.Key() == tcell.KeyCtrlZ {
		undo(edit)
		showCursor(edit)
		return false // only can return here because it is not going to change the undo/redo history
	}else if ev.Key() == tcell.KeyCtrlS {
		saveFile()
		return false
	}else if rune == 's' && alt_held {
		saveFileAs()
		return false
	}else if ev.Key() == tcell.KeyCtrlG {
		openFileByUser(filepath.Join(APP_CONFIG_DIR, "allSettings.cdmg"))
		return false
	}else if ev.Key() == tcell.KeyCtrlF {
		openFindMenu()
		return false
	}
	
	if SHOWING_INPUT_MODAL { // this is the thing... for alt+s (or general requests for text.)
		if ((ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyEsc) && INPT_TEXTEDIT.current_mode == "n") {
			INPT_TEXTEDIT.cursor.row = 0
			INPT_TEXTEDIT.cursor.col = 0
			INPT_TEXTEDIT.cursor.row_anchor = 0
			INPT_TEXTEDIT.cursor.col_anchor = 0
			INPT_TEXTEDIT.buffer = []Line{{text: ""}}
		}
		
		if ev.Key() == tcell.KeyEnter || ((ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyEsc) && INPT_TEXTEDIT.current_mode == "n") {
			SHOWING_INPUT_MODAL = false
			if SHOWING_FIND {
				CURRENT_TEXT_EDIT = "find"
				if USING_REPLACE {
					CURRENT_TEXT_EDIT = "replace"
				}
			}else{
				CURRENT_TEXT_EDIT = "main"
			}
			
			if INPUT_MODAL_CALLBACK != nil {
				INPUT_MODAL_CALLBACK()
			}
		}
	}
	
	if SHOWING_FIND {
		if (ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyEsc) && edit.current_mode == "n" {
			closeFindMenu()
			handled = true
		}
		
		if ev.Key() == tcell.KeyCtrlR || (edit.current_mode == "n" && (rune == 'r' || rune == 'f')) {
			USING_REPLACE = !USING_REPLACE
			CURRENT_TEXT_EDIT = "find"
			if USING_REPLACE {
				CURRENT_TEXT_EDIT = "replace"
			}
			handled = true
		}else if ev.Key() == tcell.KeyEnter {
			findMenuTriggered(shift_held)
			handled = true
		}
	}
	
	repeatCount := 1
	if len(edit.number_string) > 0 {
		vl, err := strconv.Atoi(edit.number_string)
		if err == nil && repeatCount != 0{
			repeatCount = vl
		}
		
	}
	
	if edit.current_mode == "n" && !handled {
		if strings.Contains(NUMBERS, string(rune)) {
			edit.number_string += string(rune)
		}else {
			edit.number_string = ""
		}
		
		if ev.Key() == tcell.KeyEnter{
			insertNewLine(edit)
		}else if rune == 'j' {
			moveCursor(MOVE_DOWN, keepAnchor, repeatCount, edit)
		}else if rune == 'k' {
			moveCursor(MOVE_UP, keepAnchor, repeatCount, edit)
		}else if rune == 'h' {
			moveCursor(MOVE_LEFT, keepAnchor, repeatCount, edit)
		}else if rune == 'l' {
			moveCursor(MOVE_RIGHT, keepAnchor, repeatCount, edit)
		}else if rune == 'e' {
			moveCursor(WORD_RIGHT, keepAnchor, repeatCount, edit)
		}else if rune == 'w' {
			moveCursor(WORD_LEFT, keepAnchor, repeatCount, edit)
		}else if ev.Key() == tcell.KeyCtrlA {
			edit.cursor.row_anchor = 0
			edit.cursor.col_anchor = 0
			moveCursor(FULL_END, true, 1, edit)
		}else if rune == 'a' {
			moveCursor(END_OF_LINE, keepAnchor, 1, edit)
			edit.current_mode = "i"
			edit.number_string = ""
			drawTitleBar()
		}else if rune == '^' {
			moveCursor(START_OF_LINE, false, 1, edit)
		}else if rune == 'o' {
			moveCursor(END_OF_LINE, false, 1, edit)
			insertNewLine(edit)
			edit.current_mode = "i"
			edit.number_string = ""
			drawTitleBar()
		}else if rune == 'v' {
			text := clipboard.Read(clipboard.FmtText)
			insertText(edit, string(text))
		}else if rune == 'c' {
			textToCopy := getCursorSelection(edit)
			clipboard.Write(clipboard.FmtText, []byte(textToCopy))
		}else if rune == 'x' {
			textToCopy := getCursorSelection(edit)
			clipboard.Write(clipboard.FmtText, []byte(textToCopy))
			deleteText(BACKSPACE, 1, edit)
		}else if rune == 'i' {
			edit.current_mode = "i"
			edit.number_string = ""
			drawTitleBar()
		}else if rune == 'g' {
			if repeatCount <= 0{
				repeatCount = 1
			}else if repeatCount > len(edit.buffer) {
				repeatCount = len(edit.buffer)
			}
			
			edit.cursor.col = 0
			edit.cursor.row = repeatCount-1
			edit.cursor.row_anchor = repeatCount-1
			edit.cursor.col_anchor = 0
			
		}else if ev.Key() == tcell.KeyDown {
			moveCursor(MOVE_DOWN, keepAnchor, 1, edit)
		}else if ev.Key() == tcell.KeyUp {
			moveCursor(MOVE_UP, keepAnchor, 1, edit)
		}else if ev.Key() == tcell.KeyLeft {
			if control_held {
				moveCursor(WORD_LEFT, keepAnchor, 1, edit)
			}else {
				moveCursor(MOVE_LEFT, keepAnchor, 1, edit)
			}
		}else if ev.Key() == tcell.KeyRight {
			if control_held {
				moveCursor(WORD_RIGHT, keepAnchor, 1, edit)
			}else {
				moveCursor(MOVE_RIGHT, keepAnchor, 1, edit)
			}
		}else if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			if control_held {
				deleteText(BACKSPACE_WORD, 1, edit)
			}else{
				deleteText(BACKSPACE, 1, edit)
			}
		}else if ev.Key() == tcell.KeyDelete {
			if control_held {
				deleteText(DELETE_WORD, 1, edit)
			}else{
				deleteText(DELETE, 1, edit)
			}
		}else if ev.Key() == tcell.KeyEnter && !SHOWING_FIND {
			insertText(edit, "\n")
		}else if rune == '\t' {
			insertText(edit, "\t")
		}else if rune == 'f' {
			openFindMenu()
		}
	}else if edit.current_mode == "i" && !handled{
		if ev.Key() == tcell.KeyEscape {
			edit.current_mode = "n"
			edit.number_string = ""
			drawTitleBar()
		}else if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			if control_held {
				deleteText(BACKSPACE_WORD, 1, edit)
			}else{
				deleteText(BACKSPACE, 1, edit)
			}
		}else if ev.Key() == tcell.KeyDelete {
			if control_held {
				deleteText(DELETE_WORD, 1, edit)
			}else{
				deleteText(DELETE, 1, edit)
			}
		}else if ev.Key() == tcell.KeyDown {
			moveCursor(MOVE_DOWN, keepAnchor, 1, edit)
		}else if ev.Key() == tcell.KeyUp {
			moveCursor(MOVE_UP, keepAnchor, 1, edit)
		}else if ev.Key() == tcell.KeyLeft {
			if control_held {
				moveCursor(WORD_LEFT, keepAnchor, 1, edit)
			}else {
				moveCursor(MOVE_LEFT, keepAnchor, 1, edit)
			}
		}else if ev.Key() == tcell.KeyRight {
			if control_held {
				moveCursor(WORD_RIGHT, keepAnchor, 1, edit)
			}else {
				moveCursor(MOVE_RIGHT, keepAnchor, 1, edit)
			}
		}else if ev.Key() == tcell.KeyEnd {
			moveCursor(END_OF_LINE, keepAnchor, 1, edit)
		}else if ev.Key() == tcell.KeyHome {
			moveCursor(START_OF_LINE, false, 1, edit)
		}else if ev.Key() == tcell.KeyEnter && !SHOWING_FIND {
			insertNewLine(edit)
		}else if ev.Key() == tcell.KeyCtrlA {
			edit.cursor.row_anchor = 0
			edit.cursor.col_anchor = 0
			moveCursor(FULL_END, true, 1, edit)
		}else {
			insertText(edit, string(rawrune))
		}
	}
	
	showCursor(edit)
	
	readyUndoHistory(edit)
	
	return false
}

func findMenuTriggered(backwards bool) {
	
}

func closeFindMenu() {
	SHOWING_FIND = false
	CURRENT_TEXT_EDIT = "main"
	redrawFullScreen()
}

func openFindMenu() {
	txt := getCursorSelection(&MAIN_TEXTEDIT)
	
	if txt != "" {
		FIND_TEXTEDIT.buffer = []Line{{text: txt, changed: true}}
		FIND_TEXTEDIT.cursor.row = len(FIND_TEXTEDIT.buffer)-1
		FIND_TEXTEDIT.cursor.col = len(FIND_TEXTEDIT.buffer[FIND_TEXTEDIT.cursor.row].text)
		FIND_TEXTEDIT.cursor.col_anchor = 0
		FIND_TEXTEDIT.cursor.row_anchor = 0
	}
	
	SHOWING_FIND = true
	CURRENT_TEXT_EDIT = "find"
	USING_REPLACE = false
	redrawFullScreen()
}

func showCursor(edit *Edit) {
	real_col := getTrueCol(edit.cursor.col, edit.cursor.row, edit)
	real_row := edit.cursor.row
	
	showing_row_start := edit.toprow-1
	showing_row_end := edit.toprow+edit.height-1
	
	showing_col_start := edit.leftchar
	sub := 0
	if edit.use_line_numbers {
		sub = len(strconv.Itoa(len(edit.buffer)))
	}
	
	showing_col_end := edit.leftchar+edit.width-sub-1 // minus 1 because cursor can be on the very end of the line.
	
	if real_col < showing_col_start {
		edit.leftchar = real_col
	}else if real_col >= showing_col_end {
		edit.leftchar += real_col-showing_col_end
	}
	
	if real_row <= showing_row_start {
		edit.toprow = real_row-7
		if edit.toprow < 0 {
			edit.toprow = 0
		}
	}else if real_row > showing_row_end {
		edit.toprow += real_row-showing_row_end+7
		if edit.toprow+edit.height > len(edit.buffer) {
			edit.toprow = len(edit.buffer)-edit.height
		}
	}
}

func copyBuffer(buffer []Line) []Line {
	copied := make([]Line, len(buffer))
	for i, line := range buffer {
		copied[i].text = line.text
		copied[i].changed = line.changed
		copied[i].styles = append([]tcell.Style{}, line.styles...)
		copied[i].end_str = line.end_str
		copied[i].end_str_type = line.end_str_type
		copied[i].start_str = line.end_str
		copied[i].start_str_type = line.end_str_type
	}
	return copied
}

func applyEditState(state Snapshot, edit *Edit) {
	edit.cursor.row = state.cursor.row
	edit.cursor.row_anchor = state.cursor.row_anchor
	edit.cursor.col = state.cursor.col
	edit.cursor.col_anchor = state.cursor.col_anchor
	edit.cursor.preferencial_col = state.cursor.preferencial_col
	
	edit.buffer = copyBuffer(state.buffer)
}

func undo(edit *Edit) {
	curText := getPlainText(edit)
	
	if len(edit.UNDO_HISTORY) > 0 {
		last_edit := edit.UNDO_HISTORY[len(edit.UNDO_HISTORY)-1]
		edit.UNDO_HISTORY = edit.UNDO_HISTORY[:len(edit.UNDO_HISTORY)-1]
		edit.REDO_HISTORY = append(edit.REDO_HISTORY, last_edit)
		
		if last_edit.fulltext == curText && len(edit.UNDO_HISTORY) > 0 {
			last_edit = edit.UNDO_HISTORY[len(edit.UNDO_HISTORY)-1]
			edit.UNDO_HISTORY = edit.UNDO_HISTORY[:len(edit.UNDO_HISTORY)-1]
			edit.REDO_HISTORY = append(edit.REDO_HISTORY, last_edit)
		}
		
		applyEditState(last_edit, edit)
	}
}

func redo(edit *Edit) {
	curText := getPlainText(edit)
	
	if len(edit.REDO_HISTORY) > 0 {
		last_edit := edit.REDO_HISTORY[len(edit.REDO_HISTORY)-1]
		edit.REDO_HISTORY = edit.REDO_HISTORY[:len(edit.REDO_HISTORY)-1]
		edit.UNDO_HISTORY = append(edit.UNDO_HISTORY, last_edit)
		
		if last_edit.fulltext == curText && len(edit.REDO_HISTORY) > 0 {
			last_edit = edit.REDO_HISTORY[len(edit.REDO_HISTORY)-1]
			edit.REDO_HISTORY = edit.REDO_HISTORY[:len(edit.REDO_HISTORY)-1]
			edit.UNDO_HISTORY = append(edit.UNDO_HISTORY, last_edit)
		}
		
		applyEditState(last_edit, edit)
	}
}

func getPlainText(edit *Edit) string {
	out := make([]string, len(edit.buffer))
	
	for indx, line := range(edit.buffer){
		out[indx] = line.text
	}
	
	return strings.Join(out, "\n")
}

func readyUndoHistory(edit *Edit) {
	cur_time_millis := time.Now().UnixNano() / 1e6
	
	overwrite := false
	
	cur_string := getPlainText(edit)
	
	if len(edit.UNDO_HISTORY) > 0 {
		last_snap := edit.UNDO_HISTORY[len(edit.UNDO_HISTORY)-1]
		if last_snap.fulltext == cur_string {
			return
		}
		
		tm := last_snap.time_taken
		
		if cur_time_millis-tm < 300 {
			overwrite = true
		}
	}
	
	copied := copyBuffer(edit.buffer)
	
	cop_cursor := Cursor{row: edit.cursor.row, col: edit.cursor.col, row_anchor: edit.cursor.row_anchor, col_anchor: edit.cursor.col_anchor, preferencial_col: edit.cursor.preferencial_col}
	
	this_snap := Snapshot{buffer: copied, cursor: cop_cursor, time_taken: cur_time_millis, fulltext: cur_string}
	
	if overwrite {
		edit.UNDO_HISTORY[len(edit.UNDO_HISTORY)-1] = this_snap
	}else{
		edit.UNDO_HISTORY = append(edit.UNDO_HISTORY, this_snap)
	}
	
	edit.REDO_HISTORY = []Snapshot{}
}

func handleKey(ev *tcell.EventKey) bool { // called in edit mode
	if SHOWING_INPUT_MODAL {
		CURRENT_TEXT_EDIT = "inpt"
		return editHandleKey(ev, &INPT_TEXTEDIT)
	}else if SHOWING_INPUT_BOOL {
		CURRENT_TEXT_EDIT = "bool"
		boolHandleKey(ev)
	}else if SHOWING_FIND {
		if USING_REPLACE {
			CURRENT_TEXT_EDIT = "replace"
			return editHandleKey(ev, &REPLACE_TEXTEDIT)
		}
		CURRENT_TEXT_EDIT = "find"
		return editHandleKey(ev, &FIND_TEXTEDIT)
	}else{
		CURRENT_TEXT_EDIT = "main"
		return editHandleKey(ev, &MAIN_TEXTEDIT)
	}
	
	return false
}

func getTrueCol(x, y int, edit *Edit) int {
	tru_col := 0
	line := edit.buffer[y]
	
	for indx := range(x){
		char := line.text[indx]
		
		if char != '\t' {
			tru_col ++
		} else {
			tru_col += 4
		}
	}
	
	return tru_col
}

func getFalseCol(x, y int, edit *Edit) int {
	fal_col := 0
	line := edit.buffer[y].text
	
	if x == 0 {
		return 0
	}
	
	for indx := range(line) {
		char := line[indx]
		
		if char == '\t' {
			pos_col := fal_col + 4
			if pos_col == x {
				return indx+1
			}else if pos_col < x {
				fal_col = pos_col
				continue
			} else { // we are around it. fal_col < x < pos_col
				if x-fal_col < pos_col-x {
					return indx
				}else{
					return indx+1
				}
			}
		}else {
			fal_col ++
			if fal_col == x {
				return indx+1
			}
		}
	}
	
	return len(line)
}

func movePointInText(x, y, action, repeat int, edit *Edit) (int, int) {
	if action == END_OF_LINE {
		x = len(edit.buffer[y].text)
	}else if action == START_OF_LINE {
		x = 0
	}else if action == FULL_END {
		y = len(edit.buffer)-1
		x = len(edit.buffer[y].text)
	}
	
	for range(repeat){
		if action == MOVE_DOWN || action == MOVE_UP {
			tru_col := getTrueCol(x, y, edit)
			
			changed := false
			if action == MOVE_DOWN && y < len(edit.buffer)-1 {
				y ++
				changed = true
			}else if action == MOVE_UP && y > 0 {
				y --
				changed = true
			}
			
			if changed {
				x = getFalseCol(tru_col, y, edit)
			}
		}
		
		if action == MOVE_LEFT {
			if x == 0 {
				if y != 0 {
					y --
					x = len(edit.buffer[y].text)
				}
			}else{
				x--
			}
		}
		
		if action == MOVE_RIGHT {
			if x == len(edit.buffer[y].text) {
				if y != len(edit.buffer)-1 {
					y ++
					x = 0
				}
			}else{
				x++
			}
		}
		
		if action == WORD_LEFT {
			if x == 0 {
				x, y = movePointInText(x, y, MOVE_LEFT, 1, edit)
				continue
			}
			
			curline := edit.buffer[y].text
			
			char := curline[x-1]
			
			strtype := getCharType(char)
			
			for range(x) {
				x -= 1
				if x == 0 {
					break
				}
				
				c := curline[x-1]
				typ := getCharType(c)
				
				if (strtype == NORMAL_CHAR_TYPE) != (typ == NORMAL_CHAR_TYPE) {
					break
				}
			}
		}
		
		if action == WORD_RIGHT {
			curline := edit.buffer[y].text
			if x == len(curline) {
				x, y = movePointInText(x, y, MOVE_RIGHT, 1, edit)
				continue
			}
			
			char := curline[x]
			
			strtype := getCharType(char)
			
			for range(len(curline)-x) {
				x += 1
				if x == len(curline) {
					break
				}
				
				c := curline[x]
				typ := getCharType(c)
				
				if (strtype == NORMAL_CHAR_TYPE) != (typ == NORMAL_CHAR_TYPE) {
					break
				}
			}
		}
	}
	
	return x, y
}

func getCharType(char byte) int {
	strype := NORMAL_CHAR_TYPE
	chr := string(char)
	if strings.Contains(WHITESPACE, chr) {
		strype = WHITESPACE_CHAR_TYPE
	}else if strings.Contains(PUNCTUATION, chr) {
		strype = PUNCTUATION_CHAR_TYPE
	}
	
	return strype
}

func moveCursor(action int, keepAnchor bool, repeat int, edit *Edit) {
	x, y := edit.cursor.col, edit.cursor.row
	
	nx, ny := movePointInText(x, y, action, repeat, edit)
	
	if action == MOVE_DOWN || action == MOVE_UP {
		nx = getFalseCol(edit.cursor.preferencial_col, ny, edit)
		if nx > len(edit.buffer[ny].text) {
			nx = len(edit.buffer[ny].text)
		}
	}
	
	edit.cursor.col = nx
	edit.cursor.row = ny
	
	if !keepAnchor {
		edit.cursor.col_anchor = edit.cursor.col
		edit.cursor.row_anchor = edit.cursor.row
	}
	
	if action == MOVE_LEFT || action == MOVE_RIGHT || action == WORD_LEFT || action == WORD_RIGHT {
		edit.cursor.preferencial_col = getTrueCol(nx, ny, edit)
	}
}

func handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()
	x, y := ev.Position()
	
	if buttons&tcell.Button1 == 0 {
		BUTTON_DOWN = false
	}
	
	if buttons&tcell.WheelUp != 0 {
		MAIN_TEXTEDIT.toprow -= SCROLL_SENSITIVITY
		if MAIN_TEXTEDIT.toprow < 0 {
			MAIN_TEXTEDIT.toprow = 0
		}
	}
	if buttons&tcell.WheelDown != 0 {
		MAIN_TEXTEDIT.toprow += SCROLL_SENSITIVITY
		if MAIN_TEXTEDIT.toprow >= len(MAIN_TEXTEDIT.buffer)-MAIN_TEXTEDIT.height {
			MAIN_TEXTEDIT.toprow = len(MAIN_TEXTEDIT.buffer)-MAIN_TEXTEDIT.height
			
			if MAIN_TEXTEDIT.toprow < 0 {
				MAIN_TEXTEDIT.toprow = 0
			}
		}
	}
	
	if buttons&tcell.Button1 != 0 {
		row := MAIN_TEXTEDIT.toprow+y-MAIN_TEXTEDIT.row
		if row >= len(MAIN_TEXTEDIT.buffer) {
			row = len(MAIN_TEXTEDIT.buffer)-1
		}else if row < 0 {
			row = 0
		}
		
		col := getFalseCol(x-len(strconv.Itoa(len(MAIN_TEXTEDIT.buffer))), row, &MAIN_TEXTEDIT)
		
		if !BUTTON_DOWN {
			MAIN_TEXTEDIT.cursor.col = col
			MAIN_TEXTEDIT.cursor.row = row
			MAIN_TEXTEDIT.cursor.col_anchor = col
			MAIN_TEXTEDIT.cursor.row_anchor = row
		}else{
			MAIN_TEXTEDIT.cursor.col = col
			MAIN_TEXTEDIT.cursor.row = row
		}
		
		BUTTON_DOWN = true
	}
	
	return false
}

func saveFile() bool {
	if file_name == "" {
		saveFileAs()
		return false // ?
	}
	
	plaintext := getPlainText(&MAIN_TEXTEDIT)
	err := os.WriteFile(file_name, []byte(plaintext), 0644)
	
	if err != nil {
		displayError("Error writing file: "+err.Error())
		return false
	}
	
	LAST_SAVED = plaintext
	
	if SAVE_CALLBACK != nil {
		SAVE_CALLBACK()
		SAVE_CALLBACK = nil
	}
	
	return true
}

func getTextInput(text string) {
	INPT_TEXTEDIT.buffer = []Line{{text: "", changed: true}} // clear old text.
	INPT_TEXTEDIT.cursor.row = 0
	INPT_TEXTEDIT.cursor.col = 0
	INPT_TEXTEDIT.cursor.row_anchor = 0
	INPT_TEXTEDIT.cursor.col_anchor = 0
	
	SHOWING_INPUT_MODAL = true
	CURRENT_TEXT_EDIT = "inpt"
	INPUT_MODAL_LABEL = text
}

func getBoolInput(text string) {
	SHOWING_INPUT_BOOL = true
	CURRENT_SELECTED_BOOL = true
	INPUT_MODAL_LABEL = text
}

func continueSaveAs() {
	new_name := getPlainText(&INPT_TEXTEDIT)
	
	if new_name == "" {
		if SAVE_CALLBACK != nil {
			SAVE_CALLBACK()
			SAVE_CALLBACK = nil
		}
		return
	}
	
	// create copy of file_name
	old_name := file_name
	file_name = new_name
	
	worked := saveFile()
	if (!worked) {
		file_name = old_name
		title = file_name
		SAVE_CALLBACK = nil
		return
	}
	
	file_name = new_name
	adjustToFileName()
	
	if SAVE_CALLBACK != nil {
		SAVE_CALLBACK()
		SAVE_CALLBACK = nil
	}
}

func saveFileAs() {
	INPUT_MODAL_CALLBACK = continueSaveAs
	getTextInput("File name?")
}

func adjustToFileName() {
	cleanedPath := filepath.Clean(file_name)
	absolute_path, _ = filepath.Abs(cleanedPath)
	title = filepath.Base(cleanedPath)
}

func openFile() {
	setupUI()
	
	file, err := os.Open(file_name)
	if err != nil {
		displayError("Error opening file: " + err.Error())
		return
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	MAIN_TEXTEDIT.buffer = []Line{}
	
	for scanner.Scan() {
		line := scanner.Text()
		MAIN_TEXTEDIT.buffer = append(MAIN_TEXTEDIT.buffer, Line{text: line, changed: true})
	}
	
	LAST_SAVED = getPlainText(&MAIN_TEXTEDIT)

	if err := scanner.Err(); err != nil {
		displayError("Error reading lines: " + err.Error())
	}
	
	adjustToFileName()
	getSavedPlace()
}

func openFileByUser(file_to_open string) {
	opening_file = file_to_open
	
	if LAST_SAVED != getPlainText(&MAIN_TEXTEDIT) {
		CHECK_FOR_SAVE_CALLBACK = nextOpenFileByUser
		checkForSave()
	}else{
		nextOpenFileByUser()
	}
}

func closeOnCheckDone() {
	SAVE_CALLBACK = nil
	CHECK_FOR_SAVE_CALLBACK = nil
	NEED_TO_EXIT = true
}

func nextOpenFileByUser() {
	CHECK_FOR_SAVE_CALLBACK = nil
	SAVE_CALLBACK = nil
	
	file_name = opening_file
	openFile()
}

func checkForSave() {
	INPUT_MODAL_CALLBACK = continueCheckForSave
	getBoolInput("Unsaved, changes save?")
}


func continueCheckForSave() {
	if CURRENT_SELECTED_BOOL {
		SAVE_CALLBACK = CHECK_FOR_SAVE_CALLBACK
		saveFile()
	}else{
		if CHECK_FOR_SAVE_CALLBACK != nil {
			CHECK_FOR_SAVE_CALLBACK()
		}
	}
}

func displayError(errorMessage string) {
	SHOWING_INPUT_MODAL = true
	CURRENT_TEXT_EDIT = "inpt"
	
	INPT_TEXTEDIT.cursor.row = 0
	INPT_TEXTEDIT.cursor.col = 0
	INPT_TEXTEDIT.cursor.row_anchor = 0
	INPT_TEXTEDIT.cursor.col_anchor = 0
	
	INPUT_MODAL_LABEL = ""
	INPUT_MODAL_CALLBACK = nil
	INPT_TEXTEDIT.buffer = []Line{{text: errorMessage, changed: true}}
}

func getConfigDir() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error getting user config directory: %v", err)
	}
	APP_CONFIG_DIR = filepath.Join(configDir, "CodeMage")
}

func getSpecificVar(known [][]string, lookingfor string) string {
	for i := range(known) {
		if len(known[i]) != 2 {
			continue
		}
		
		if known[i][0] == lookingfor {
			return known[i][1]
		}
	}
	
	return ""
}

func getTcellColor(inpt string, def tcell.Color) tcell.Color {
	inpt = strings.ReplaceAll(inpt, " ", "")
	rgb := strings.Split(inpt, ",")
	
	if len(rgb) != 3 {
		return def
	}
	
	nrgb := []int32{}
	
	for _, v := range(rgb) {
		val, err := strconv.Atoi(v)
		if err != nil || val > 255 || val < 0 {
			return def
		}
		nrgb = append(nrgb, int32(val))
	}
	
	return tcell.NewRGBColor(nrgb[0], nrgb[1], nrgb[2])
}

func getInt(found string, def int) int {
	vl, err := strconv.Atoi(found)
	if err != nil {
		return def
	}
	
	return vl
}

func loadSettings() {
	settings_path := filepath.Join(APP_CONFIG_DIR, "allSettings.cdmg")
	_, err := os.Stat(settings_path);
	
	if os.IsNotExist(err) {
		return
	}
	
	file, err := os.Open(settings_path)
	if err != nil {
		return
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	known := [][]string{}
	
	for scanner.Scan() {
		line := scanner.Text()
		
		splt := strings.Split(line, ": ")
		known = append(known, splt)
	}
	
	if err := scanner.Err(); err != nil {
		displayError("Error reading lines: " + err.Error())
	}
	
	colorSTRING = getTcellColor(getSpecificVar(known,"colorSTRING"), tcell.NewRGBColor(127, 173, 94))
	colorFUNCTION = getTcellColor(getSpecificVar(known,"colorFUNCTION"), tcell.NewRGBColor(199, 157, 78))
	colorKEYWORD = getTcellColor(getSpecificVar(known,"colorKEYWORD"), tcell.NewRGBColor(176, 95, 199))
	colorNAME = getTcellColor(getSpecificVar(known,"colorNAME"), tcell.NewRGBColor(245, 91, 102))
	colorPUNC = getTcellColor(getSpecificVar(known,"colorPUNC"), tcell.NewRGBColor(127, 132, 142))
	colorCOMMENT = getTcellColor(getSpecificVar(known,"colorCOMMENT"), tcell.NewRGBColor(127, 132, 142))
	colorLITTERAL = getTcellColor(getSpecificVar(known,"colorLITTERAL"), tcell.NewRGBColor(194, 127, 64))
	SCROLL_SENSITIVITY = getInt(getSpecificVar(known,"SCROLL_SENSITIVITY"), 3)
}

func getcolorSTRING(col tcell.Color) string {
	r, g, b := col.RGB()
	return strconv.Itoa(int(r))+", "+strconv.Itoa(int(g))+", "+strconv.Itoa(int(b))
}

func savePlace() {
	settings_path := filepath.Join(APP_CONFIG_DIR, "savedPlaces.cdmg")
	_, err := os.Stat(settings_path);
	
	savedPlaces := []string{}
	
	if err == nil {
		file, err := os.Open(settings_path)
		if err == nil {
			defer file.Close()
			
			scanner := bufio.NewScanner(file)
			
			for scanner.Scan() {
				line := scanner.Text()
				
				splt := strings.Split(line, " ! ")
				if len(splt) == 2 {
					if splt[0] != absolute_path {
						savedPlaces = append(savedPlaces, line)
					}
				}
			}
		}
	}
	
	savedPlaces = append(savedPlaces, absolute_path+" ! "+strconv.Itoa(MAIN_TEXTEDIT.cursor.row)+","+strconv.Itoa(MAIN_TEXTEDIT.cursor.col)+","+strconv.Itoa(MAIN_TEXTEDIT.cursor.row_anchor)+","+strconv.Itoa(MAIN_TEXTEDIT.cursor.col_anchor))
	
	os.WriteFile(settings_path, []byte(strings.Join(savedPlaces, "\n")), 0644)
}

func getSavedPlace() {
	settings_path := filepath.Join(APP_CONFIG_DIR, "savedPlaces.cdmg")
	_, err := os.Stat(settings_path);
	
	if os.IsNotExist(err) {
		return
	}
	
	file, err := os.Open(settings_path)
	if err != nil {
		return
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		splt := strings.Split(line, " ! ")
		if len(splt) == 2 {
			if splt[0] == absolute_path {
				splt2 := strings.Split(splt[1], ",")
				if len(splt2) != 4 {
					continue
				}
				
				nums := []int{}
				for _, nS := range splt2 {
					num, err := strconv.Atoi(nS)
					if err != nil {
						continue
					}
					nums = append(nums, num)
				}
				
				if nums[0] < 0{
					nums[0] = 0
				}else if nums[0] >= len(MAIN_TEXTEDIT.buffer) {
					nums[0] = len(MAIN_TEXTEDIT.buffer)-1
				}
				
				if nums[2] < 0{
					nums[2] = 0
				}else if nums[2] >= len(MAIN_TEXTEDIT.buffer) {
					nums[2] = len(MAIN_TEXTEDIT.buffer)-1
				}
				
				if nums[1] < 0{
					nums[1] = 0
				}else if nums[1] > len(MAIN_TEXTEDIT.buffer[nums[0]].text) {
					nums[1] = len(MAIN_TEXTEDIT.buffer[nums[0]].text)
				}
				
				if nums[3] < 0{
					nums[3] = 0
				}else if nums[3] > len(MAIN_TEXTEDIT.buffer[nums[2]].text) {
					nums[3] = len(MAIN_TEXTEDIT.buffer[nums[2]].text)
				}
				
				MAIN_TEXTEDIT.cursor.row = nums[0]
				MAIN_TEXTEDIT.cursor.col = nums[1]
				MAIN_TEXTEDIT.cursor.row_anchor = nums[2]
				MAIN_TEXTEDIT.cursor.col_anchor = nums[3]
				showCursor(&MAIN_TEXTEDIT)
				return
			}
		}
	}
}

func saveSettings() {
	_, err := os.Stat(APP_CONFIG_DIR);
	
	if os.IsNotExist(err) {
		err = os.Mkdir(APP_CONFIG_DIR, 0755)
	}
	
	settings_path := filepath.Join(APP_CONFIG_DIR, "allSettings.cdmg")
	
	settings_lines := []string{}
	
	settings_lines = append(settings_lines, "Syntax colors for syntax highligting in RGB")
	settings_lines = append(settings_lines, "colorSTRING: "+getcolorSTRING(colorSTRING))
	settings_lines = append(settings_lines, "colorFUNCTION: "+getcolorSTRING(colorFUNCTION))
	settings_lines = append(settings_lines, "colorKEYWORD: "+getcolorSTRING(colorKEYWORD))
	settings_lines = append(settings_lines, "colorNAME: "+getcolorSTRING(colorNAME))
	settings_lines = append(settings_lines, "colorPUNC: "+getcolorSTRING(colorPUNC))
	settings_lines = append(settings_lines, "colorCOMMENT: "+getcolorSTRING(colorCOMMENT))
	settings_lines = append(settings_lines, "colorLITTERAL: "+getcolorSTRING(colorLITTERAL))
	settings_lines = append(settings_lines, "\nDecreasing scroll sensitivity helps make the scrolling look better (lesser changes), but it must be an int >= 0.")
	settings_lines = append(settings_lines, "SCROLL_SENSITIVITY: "+strconv.Itoa(SCROLL_SENSITIVITY))
	
	os.WriteFile(settings_path, []byte(strings.Join(settings_lines, "\n")), 0644)
}

func main() {
	getConfigDir()
	loadSettings()
	saveSettings()
	
	if len(os.Args) > 1 {
		file_name = os.Args[1]
		cleanedPath := filepath.Clean(file_name)
		absPath, err := filepath.Abs(cleanedPath)
		absolute_path = absPath
		
		if err != nil {
			log.Fatalf("%+v", err)
		}
		
		fileInfo, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			fmt.Printf("File does not exist\n")
			return
		} else if err != nil {
			fmt.Printf("Error checking existence: %v\n", err)
			return
		} else {
			if fileInfo.IsDir() {
				fmt.Printf("Specified file was a directory.")
				return
			}
		}

		title = filepath.Base(cleanedPath)
	}
	
	err := clipboard.Init()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	
	s, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	defer s.Fini()
	
	s.EnableMouse()
	
	DEF_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(tcell.ColorWhite)
	INVERTED_STYLE = tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	TITLE_STYLE = tcell.StyleDefault.Background(titleColor).Foreground(tcell.ColorWhite)
	HIGHLIGHT_STYLE = tcell.StyleDefault.Background(highlightColor).Foreground(tcell.ColorWhite)
	LINE_NUMBER_STYLE = tcell.StyleDefault.Background(lineNumberColor).Foreground(tcell.ColorWhite)
	STRING_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorSTRING)
	NORMAL_MODE_STYLE = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorBlack)
	FUNCTION_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorFUNCTION)
	KEYWORD_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorKEYWORD)
	NAME_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorNAME)
	PUNC_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorPUNC)
	COMMENT_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorCOMMENT)
	LITTERAL_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorLITTERAL)
	SPECIAL_STYLE = tcell.StyleDefault.Background(colorBackground).Foreground(colorSpecial)
	
	s.SetStyle(DEF_STYLE)
	s.Clear()
	s.HideCursor()
	
	if file_name == ""{
		current_window = "blank"
		redrawFullScreen()
	}else{
		current_window = "edit"
		openFile()
		
	}
	
	for {
		ev := s.PollEvent()
		
		switch ev := ev.(type) {
		case *tcell.EventKey:
//			if ev.Key() == tcell.KeyEscape {
//				return
//			}
			
			if current_window != "edit"{
				current_window = "edit"
				createNew()
			}
			
			if handleKey(ev) { // exit condition
				if LAST_SAVED == getPlainText(&MAIN_TEXTEDIT) {
					savePlace()
					return
				}
				CHECK_FOR_SAVE_CALLBACK = closeOnCheckDone
				checkForSave()
			}
			
			
			drawFullEdit()
		case *tcell.EventMouse:
			if current_window == "edit" {
				if handleMouse(ev) {
					if LAST_SAVED == getPlainText(&MAIN_TEXTEDIT) {
						savePlace()
						return
					}
					CHECK_FOR_SAVE_CALLBACK = closeOnCheckDone
					checkForSave()
				}
				
				drawFullEdit()
			}
		case *tcell.EventResize:
			redrawFullScreen()
		
		default:
			// You can choose to log or ignore other event types
		}
		
		if NEED_TO_EXIT {
			savePlace()
			return // this is the exit condition
		}
		
		s.Show()
	}
}