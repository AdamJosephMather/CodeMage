# CodeMage - Terminal-Based Text Editor

<img width="1128" alt="image" src="https://github.com/user-attachments/assets/7f12bdcb-a6bf-4d0c-899f-2aa3c49e9cc2" />

## Overview

CodeMage is a terminal-based text editor developed in Go, designed to provide an efficient, lightweight, and customizable editing experience within the terminal.  It serves as a streamlined counterpart to CodeWizard, offering core editing features without the overhead of a graphical interface. 

## Features

Syntax Highlighting: Real-time syntax highlighting for popular programming languages.

Undo/Redo Functionality: Easily revert and reapply changes.

Modal Editing: Inspired by Vim, includes modal editing modes for efficient text manipulation.

Clipboard Integration: Seamless integration with the system clipboard for copying and pasting text.

Search and Replace: Built-in find and replace functionality.

Customizable Styles: Adjust color themes and text styles to enhance readability. 


## Installation

1. Clone the repository:

git clone https://github.com/AdamJosephMather/CodeMage.git
cd CodeMage


2. Install dependencies:

go get

3. Build and run:

go build -o cdmg

cdmg

or

cdmg [filename]

3. Enjoy

Navigate and edit using keyboard commands.

Switch modes for different editing functionalities.

Save, undo, redo, and manage files directly from the terminal.


## Shortcuts

Ctrl + S: Save file.

Ctrl + Z: Undo.

Ctrl + Y: Redo.

Ctrl + F: Find text.

(insert all of the others here - I know nobody's using this but me...)

Esc: Switch modes (command/edit). 

CodeMage's modal experience is extremely similar to CodeWizard's 'VIM' mode.

Relationship to CodeWizard

CodeMage is developed as a lighter weight CodeWizard (a feature-rich, Qt-based code editor and IDE)  While CodeWizard offers a comprehensive graphical interface with advanced features like LSP support, AI integration, and theming, CodeMage focuses on delivering a minimalist, terminal-based editing experience.  It captures the essence of efficient text editing, making it ideal for users who prefer or require a terminal environment. 
