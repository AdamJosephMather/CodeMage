package main

import (
//	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type Edit struct {
	row int
	col int
	width int
	height int
	
	buffer []string
	
	toprow int
	leftchar int
	cursor Cursor
	use_line_numbers bool
	
	current_mode string
}

type Cursor struct {
	row int
	col int
	row_anchor int
	col_anchor int
}

var version string = "0.0.1"

var s tcell.Screen
var current_window string
var file_name string
var title string

var defStyle tcell.Style
var invertedStyle tcell.Style
var titleStyle tcell.Style
var highlightStyle tcell.Style
var lineNumberStyle tcell.Style

var MAIN_TEXTEDIT Edit

var MOVE_DOWN = 1
var MOVE_UP = 2
var MOVE_LEFT = 3
var MOVE_RIGHT = 4

var BACKSPACE = 5
var DELETE = 6

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for i, r := range []rune(str) {
		s.SetContent(x+i, y, r, nil, style)
	}
}

func emitStrColored(s tcell.Screen, x, y int, style []tcell.Style, str string) {
	for i, r := range []rune(str) {
		s.SetContent(x+i, y, r, nil, style[i])
	}
}

func mouseButtonsToString(buttons tcell.ButtonMask) string {
	var s []string
	if buttons&tcell.Button1 != 0 {
		s = append(s, "Left")
	}
	if buttons&tcell.Button3 != 0 {
		s = append(s, "Middle")
	}
	if buttons&tcell.Button2 != 0 {
		s = append(s, "Right")
	}
	if buttons&tcell.Button4 != 0 { // Often scroll up
		s = append(s, "ScrollUp")
	}
	if buttons&tcell.Button5 != 0 { // Often scroll down
		s = append(s, "ScrollDown")
	}
	if buttons&tcell.ButtonPrimary != 0 {
		s = append(s, "Primary")
	}
	if buttons&tcell.ButtonSecondary != 0 {
		s = append(s, "Secondary")
	}
	if buttons&tcell.ButtonNone != 0 && len(s) == 0 { // No buttons pressed (e.g., mouse movement)
		s = append(s, "None")
	}
	return strings.Join(s, ", ")
}

func createNew(s tcell.Screen) {
	width, height := s.Size()
	
	title = "Untitled"
	file_name = ""
	
	buffer := make([]string, 1)
	
	cursor := Cursor{row: 0, col: 0, row_anchor: 0, col_anchor: 0}
	buffer[0] = ""
	
	MAIN_TEXTEDIT = Edit{row: 1, col: 0, width: width, height: height-1, buffer: buffer, cursor: cursor, toprow: 0, leftchar: 0, use_line_numbers: true, current_mode: "i"}
	
	redrawFullScreen(s)
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

func drawEdit(s tcell.Screen, edit Edit) {
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
			emitStr(s, edit.col, y, lineNumberStyle, strings.Repeat(" ", line_num_width-1)+"~"+strings.Repeat(" ", edit.width-line_num_width))
			emitStr(s, edit.col+line_num_width, y, defStyle, strings.Repeat(" ", edit.width-line_num_width))
			continue
		}else if line_num >= len(buffer){
			emitStr(s, edit.col, y, defStyle, strings.Repeat(" ", edit.width))
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
			
			emitStr(s, edit.col, y, lineNumberStyle, fullstr)
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
				maxRng = len(edit.buffer[line_num])+1
			}else if line_num == start_row && line_num == end_row {
				minRng = start_col-1
				maxRng = end_col
			}else if line_num == start_row {
				minRng = start_col-1
				maxRng = len(edit.buffer[line_num])+1
			}else if line_num == end_row {
				minRng = -1
				maxRng = end_col
			}
		}
				
		lineToDraw := ""
		
		runes := []rune(buffer[line_num])
		xraw := 0
		
		curs_line := cursor_pos == line_num
		curs_char := cursor.col
		
		styles := []tcell.Style{}
		
		for true {
			charIndx := edit.leftchar + xraw
			
			is_cursor := curs_line && charIndx == curs_char
			is_in_highlight := charIndx > minRng && charIndx < maxRng
			
			cur_style := defStyle
			
			if is_cursor {
				cur_style = invertedStyle
			}else if is_in_highlight {
				cur_style = highlightStyle
			}
						
			if charIndx >= len(runes) {
				lineToDraw += strings.Repeat(" ", edit.width-len(lineToDraw))
				styles = append(styles, cur_style)
				styles = append(styles, repeatSlice(defStyle, edit.width-len(styles))...)
				break
			}
			
			char := runes[charIndx]
			
			if char != '\t'{
				lineToDraw += string(char)
				styles = append(styles, cur_style)
			}else{
				for range(4) {
					lineToDraw += " "
					styles = append(styles, defStyle)
					if len(lineToDraw) == edit.width {break}
					
				}
				styles[len(styles)-1] = cur_style
			}
			
			xraw ++
			
			if len(lineToDraw) == edit.width {
				break
			}
		}
		
		x := edit.col+line_num_width
		
		if len(styles) != len(lineToDraw) {
			emitStr(s, x, y, defStyle, "Len no matchy "+strconv.Itoa(len(styles)))
		}else{
			emitStrColored(s, x, y, styles, lineToDraw)
		}
	}
}

func drawFullEdit(s tcell.Screen) {
	drawEdit(s, MAIN_TEXTEDIT)
	drawTitleBar(s)
}

func drawTitleBar(s tcell.Screen) {
	w, _ := s.Size()
	
	text := "CodeMage V"+version+" - "+title
	text += strings.Repeat(" ", w-len(text))
	emitStr(s, 0, 0, titleStyle, text)
	
	
	text = "ERROR IN MAKING THE TITLEBAR?"
	if MAIN_TEXTEDIT.current_mode == "n" {
		text = "NORMAL"
	}else if MAIN_TEXTEDIT.current_mode == "i" {
		text = "INSERT"
	}
	emitStr(s, w-len(text), 0, titleStyle, text)
}

func redrawFullScreen(s tcell.Screen) {
	s.Sync()
	s.Clear()
	width, height := s.Size()
	
	if current_window == "blank"{
		lines := []string{"CodeMage V"+version, "Designed by Adam Mather"}
		
		for i := range(lines){
			line := lines[i]
			linelen := len(line)
			startX := (width-linelen)/2
			startY := (height/2)+i
			
			emitStr(s, startX, startY, defStyle, line)
		}
	}else if current_window == "edit" {
		MAIN_TEXTEDIT.width = width
		MAIN_TEXTEDIT.height = height-1
		
		drawFullEdit(s)
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
			
			start := line[:start_col]
			end := line[end_col:]
			
			edit.buffer[start_row] = start + end // delete section
		}else{
			first_line := edit.buffer[start_row][:start_col]
			last_line := edit.buffer[end_row][end_col:]
			edit.buffer[start_row] = first_line + last_line
			
			edit.buffer = append(edit.buffer[:start_row+1], edit.buffer[end_row+1:]...)
		}
		
		edit.cursor.row = start_row
		edit.cursor.col = start_col
		edit.cursor.row_anchor = start_row
		edit.cursor.col_anchor = start_col
		
		return
	}
	
	for range(repeat){
		if mode == BACKSPACE {
			line := edit.buffer[edit.cursor.row]
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
				
				joining_line := edit.buffer[edit.cursor.row]
				
				edit.cursor.col = len(joining_line)
				
				edit.buffer[edit.cursor.row] += after_deleted
				edit.buffer = append(edit.buffer[:edit.cursor.row+1], edit.buffer[edit.cursor.row+2:]...)
			}else{
				moveCursor(MOVE_LEFT, false, 1, edit)
				edit.buffer[edit.cursor.row] = before_deleted+after_deleted
			}
		}
	}
}

func insertText(edit *Edit, text string) {
	if edit.cursor.row != edit.cursor.row_anchor || edit.cursor.col != edit.cursor.col_anchor {
		deleteText(BACKSPACE, 1, edit) // clear selection
	}
	
	lines := strings.Split(text, "\n")
	
	first_len := len(lines[0])
	end_len := len(lines[len(lines)-1])
	
	current_line := edit.buffer[edit.cursor.row]
	bfrCrs := current_line[:edit.cursor.col]
	endCrs := current_line[edit.cursor.col:]
	
	lines[0] = bfrCrs + lines[0]
	lines[len(lines)-1] = lines[len(lines)-1] + endCrs
	
	edit.buffer[edit.cursor.row] = lines[0]
	
	end_line := edit.cursor.row
	end_char := edit.cursor.col + first_len
	
	buffer := edit.buffer
	
	if len(lines) > 1 {
		insertionIndex := edit.cursor.row+1
		
		start := append([]string(nil), buffer[:insertionIndex]...) // ensure copy not reference
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
}

func editHandleKey(s tcell.Screen, ev *tcell.EventKey, edit *Edit) {
	rawrune := ev.Rune()
	
	rune := unicode.ToLower(rawrune)
	
	
	if edit.current_mode == "n" {
		keepAnchor := false
		
		if ev.Modifiers() & tcell.ModShift != 0{
			keepAnchor = true
		}
		
		if ev.Key() == tcell.KeyEnter{
			insertText(edit, "\n")
		}else if rune == 'j' {
			moveCursor(MOVE_DOWN, keepAnchor, 1, edit)
		}else if rune == 'k' {
			moveCursor(MOVE_UP, keepAnchor, 1, edit)
		}else if rune == 'h' {
			moveCursor(MOVE_LEFT, keepAnchor, 1, edit)
		}else if rune == 'l' {
			moveCursor(MOVE_RIGHT, keepAnchor, 1, edit)
		}else if rune == 'i' {
			edit.current_mode = "i"
			drawTitleBar(s)
		}else if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			deleteText(BACKSPACE, 1, edit)
		}
	}else if edit.current_mode == "i"{
		if ev.Key() == tcell.KeyEscape {
			edit.current_mode = "n"
			drawTitleBar(s)
		}else if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			deleteText(BACKSPACE, 1, edit)
		}else if ev.Key() == tcell.KeyEnter {
			insertText(edit, "\n")
		}else {
			insertText(edit, string(rawrune))
		}
	}
	
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
		edit.toprow = real_row
	}else if real_row >= showing_row_end {
		edit.toprow += real_row-showing_row_end
	}
}

func handleKey(s tcell.Screen, ev *tcell.EventKey){ // called in edit mode
	editHandleKey(s, ev, &MAIN_TEXTEDIT)
}

func getTrueCol(x, y int, edit *Edit) int {
	tru_col := 0
	line := edit.buffer[y]
	
	for indx := range(x){
		char := line[indx]
		
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
	line := edit.buffer[y]
	
	if x == 0 {
		return 0
	}
	
	for indx := range(line) {
		char := line[indx]
		
		if char == '\t'{
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
					x = len(edit.buffer[y])
				}
			}else{
				x--
			}
		}
		
		if action == MOVE_RIGHT {
			if x == len(edit.buffer[y]) {
				if y != len(edit.buffer)-1 {
					y ++
					x = 0
				}
			}else{
				x++
			}
		}
	}
	
	return x, y
}

func moveCursor(action int, keepAnchor bool, repeat int, edit *Edit) {
	x, y := edit.cursor.col, edit.cursor.row
	nx, ny := movePointInText(x, y, action, repeat, edit)
	edit.cursor.col = nx
	edit.cursor.row = ny
	
	if !keepAnchor {
		edit.cursor.col_anchor = edit.cursor.col
		edit.cursor.row_anchor = edit.cursor.row
	}
}

func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	defer s.Fini()
	
	s.EnableMouse()
	
	titleColor := tcell.NewRGBColor(25, 25, 25)
	highlightColor := tcell.NewRGBColor(100, 100, 100)
	lineNumberColor := tcell.NewRGBColor(42, 42, 42)
	
	defStyle = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	invertedStyle = tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	titleStyle = tcell.StyleDefault.Background(titleColor).Foreground(tcell.ColorWhite)
	highlightStyle = tcell.StyleDefault.Background(highlightColor).Foreground(tcell.ColorWhite)
	lineNumberStyle = tcell.StyleDefault.Background(lineNumberColor).Foreground(tcell.ColorWhite)
	
	s.SetStyle(defStyle)
	s.Clear()
	s.HideCursor()
	
	current_window = "blank"
	
	redrawFullScreen(s)
	
	for {
		ev := s.PollEvent()
		
		switch ev := ev.(type) {
		case *tcell.EventKey:
//			if ev.Key() == tcell.KeyEscape {
//				return
//			}
			
			if current_window == "edit"{
				handleKey(s, ev)
			}else{
				current_window = "edit"
				createNew(s)
			}
			
			drawFullEdit(s)
//		case *tcell.EventMouse:
//			x, y := ev.Position()
//			buttons := ev.Buttons()
//			emitStr(s, 0, 0, defStyle, fmt.Sprintf("(%d, %d) - %s", x, y, mouseButtonsToString(buttons)))
		
		case *tcell.EventResize:
			redrawFullScreen(s)
		
		default:
			// You can choose to log or ignore other event types
		}
		s.Show()
		time.Sleep(10 * time.Millisecond)
	}
}