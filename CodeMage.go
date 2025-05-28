package main

import (
	"fmt"
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

var defStyle tcell.Style

var textedit Edit

var MOVE_DOWN = 1
var MOVE_UP = 2
var MOVE_LEFT = 3
var MOVE_RIGHT = 4

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for i, r := range []rune(str) {
		s.SetContent(x+i, y, r, nil, style)
	}
}

func mouseButtonsToString(buttons tcell.ButtonMask) string {
	var s []string
	if buttons&tcell.Button1 != 0 {
		s = append(s, "Left")
	}
	if buttons&tcell.Button2 != 0 {
		s = append(s, "Middle")
	}
	if buttons&tcell.Button3 != 0 {
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
	
	buffer := make([]string, 1)
	
	cursor := Cursor{row: 0, col: 0, row_anchor: 0, col_anchor: 0}
	buffer[0] = ""
	
	textedit = Edit{row: 1, col: 0, width: width, height: height-1, buffer: buffer, cursor: cursor, toprow: 0, leftchar: 0, use_line_numbers: true}
	
	redrawFullScreen(s)
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
		
		if line_num >= len(buffer) {
			emitStr(s, edit.col, y, defStyle, strings.Repeat(" ", line_num_width-1)+"~")
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
			
			emitStr(s, edit.col, y, defStyle, fullstr)
		}
		
		lineToDraw := ""
		
		runes := []rune(buffer[line_num])
		xraw := 0
		
		for true {
			charIndx := edit.leftchar + xraw
			
			if charIndx >= len(runes) {
				lineToDraw += strings.Repeat(" ", edit.width-len(lineToDraw))
				break
			}
			
			char := runes[charIndx]
			
			if char != '\t'{
				lineToDraw += string(char)
				xraw ++
			}else{
				for range(4) {
					lineToDraw += " "
					xraw ++
					if xraw == edit.width-line_num_width {break}
				}
			}
			if xraw == edit.width-line_num_width {
				break
			}
		}
		x := edit.col+line_num_width
		emitStr(s, x, y, defStyle, lineToDraw)
	}
}

func drawFullEdit(s tcell.Screen) {
	drawEdit(s, textedit)
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
		textedit.width = width
		textedit.height = height-1
		
		drawFullEdit(s)
	}
	
	s.Show()
}

func editHandleKey(s tcell.Screen, ev *tcell.EventKey, edit *Edit) {
	rawrune := ev.Rune()
	
	rune := unicode.ToLower(rawrune)
	keepAnchor := false
	
	if rune != rawrune {
		keepAnchor = true
	}
	
	if ev.Key() == tcell.KeyEnter{
		line := edit.cursor.row+1
		edit.buffer = append(edit.buffer[:line], append([]string{""}, edit.buffer[line:]...)...)
		moveCursor(MOVE_DOWN, keepAnchor, 1, edit)
	}
	
	if rune == 'j' {
		moveCursor(MOVE_DOWN, keepAnchor, 1, edit)
	}
	
	if rune == 'k' {
		moveCursor(MOVE_UP, keepAnchor, 1, edit)
	}
}

func handleKey(s tcell.Screen, ev *tcell.EventKey){ // called in edit mode
	editHandleKey(s, ev, &textedit)
}

func movePointInText(x, y, action, repeat int, textedit *Edit) (int, int) {
	for range(repeat){
		if action == MOVE_DOWN && y < len(textedit.buffer)-1 {
			y ++
		}else if action == MOVE_UP && y > 0 {
			y --
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
		x, y := edit.cursor.col_anchor, edit.cursor.row_anchor
		nx, ny := movePointInText(x, y, action, repeat, edit)
		edit.cursor.col_anchor = nx
		edit.cursor.row_anchor = ny
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
	
	defStyle = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)
	s.Clear()
	s.HideCursor()
	
	current_window = "blank"
	
	redrawFullScreen(s)
	
	for {
		ev := s.PollEvent()
		
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				return
			} // to test git push once again again
			
			if current_window == "edit"{
				handleKey(s, ev)
				emitStr(s, 0, 0, defStyle, ev.Name())
			}else{
				current_window = "edit"
				createNew(s)
			}
			
			drawFullEdit(s)
		case *tcell.EventMouse:
			x, y := ev.Position()
			buttons := ev.Buttons()
			emitStr(s, 0, 0, defStyle, fmt.Sprintf("(%d, %d) - %s", x, y, mouseButtonsToString(buttons)))
		
		case *tcell.EventResize:
			redrawFullScreen(s)
		
		default:
			// You can choose to log or ignore other event types
		}
		s.Show()
		time.Sleep(10 * time.Millisecond)
	}
}