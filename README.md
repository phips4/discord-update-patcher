# discord-update-patcher
tries to fix the discord update loop issue

First of all, its technically not a patcher, it just tries to update your discord where the
discord updater fails and got stuck.

I created this program because I got this bug at least every second update discord 
released even on different machines at different times. After a little of reverse 
engineering on the discord updater and found what is going wrong. Fixing it manually
everytime was very time intense and annoying so, I decided to automate this progress.

It is only intended for windows, because the bug seems only to occur on windows.

## How to use
Download a binary version from github releases or compile it by yourself. 
Execute the executable like following:
```zsh
./discord-update-patcher.exe
```

## TODO
    * [x] baisc update logic
    * [ ] backup reverting
    * [ ] check if discord is running