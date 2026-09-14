Output demo.gif 

Set Shell "zsh" 
Set FontSize 28 
Set Columns 195 
Set Rows 45 
Set Theme "Kanagawa" 
Set TypingSpeed 45ms 
Set Margin 32 
Set MarginFill "#223249" 
Set BorderRadius 16

# open app
Type@25ms "../qrypad --connection demo"
Enter
Sleep 750ms

# show some table data
Enter
Sleep 750ms

# navigate to query panel
Tab
Tab

# enter a query with auto complete
Enter
Enter
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
Sleep 750ms

# to results
Tab
Sleep 600ms
Type "/Buster"
Sleep 800ms
Escape
Down
Sleep 600ms

# results popup
Enter
Down
Sleep 600ms
Down
Sleep 600ms
Up
Sleep 600ms
Escape
Sleep 750ms
