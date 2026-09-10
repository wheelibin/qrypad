# Generate: vhs demo.tape
# Requires: go build -o qrypad .   (run from repo root first)
#
# Runs against ./demo/config.toml, a standalone config that points at the
# committed test-db/sqlite/sqlite.db and remaps ExecuteQuery from F5 to
# Ctrl+G — vhs 0.12.0 has no function-key command, so F5 can't be sent.

Output demo.gif

Set Shell "zsh"
Set FontSize 28 
Set Columns 195
Set Rows 45 
Set Theme "Kanagawa"
Set TypingSpeed 90ms

Set Margin 32 
Set MarginFill "#223249"
Set BorderRadius 16

# open app
Type@30ms "../qrypad --connection demo"
Enter
Sleep 2s

# show some table data
Enter
Sleep 1s
Down
Enter
Sleep 1s
Down
Enter
Sleep 1s

# navigate to query panel
Tab
Tab

# enter a query with auto complete
Type "select *"
Enter
Type "from "
Down
Enter
Type " a"
Enter
Type "join "
Down
Enter
Type " s on s.id"
Enter
Type " = a.spec"
Enter
Enter
Type "where s.name"
Enter
Type " = 'Dog';"
Up
Up
Up
Backspace
Type "a.name, a.age, a.health_records"
Enter
Ctrl+G
Sleep 2s

# to results
Tab
Sleep 1s
Type "/Buster"
Sleep 1s
Escape
Down
Sleep 1s

# results popup
Enter
Down
Sleep 1s
Down
Sleep 1s
Up
Sleep 1s
Escape
Sleep 2s
