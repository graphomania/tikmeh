## Tikmeh
#### 0.1.0 (Sep 26, 2022)

Single executable to **download videos, profiles, sync your collection** with authors in one command with the best
quality available.
No installation required, you don't have to use Terminal.

I was asked to add conversion to `h.264`, so now Tikmeh could access user-provided `ffmpeg`, by default the system's one.

- tikmeh.exe – Windows (amd64) executable (compatible even with Win7)
- tikmeh – Linux (amd64) executable

### Examples:

- `tikmeh`  – interactive mode (more on that later)
- `tikmeh tiktok.com/@shrimpydimpy/video/7133412834960018730`  – simply download the video
- `tikmeh profile @shrimpydimpy @losertron`                    – download all @shrimpydimpy, @losertron
  videos to `./shrimpydimpy`, `./losertron` accordingly
- `tikmeh directory ./mp4 profile @shrimpydimpy`            – download all @shrimpydimpy videos to `./mp4`
- `tikmeh profile @shrimpydimpy convert ffmpeg-path ./ffmpeg.exe`    – download and convert videos to h264 with `./ffmpeg.exe`
- `tikmeh profile @shrimpydimpy check-all`          – download all missing videos, ensure every video is downloaded

### Sync?

Yes, literally synchronization. Just download a profile once in full and Tikmeh wouldn't re-upload already downloaded
videos.  

Note: by default Tikmeh loads the profile until it meets already downloaded video, 
to ensure nothing is skipped, use `check-all` flag.
By default, directory named after the profile username is created.

### Interactive mode

Exists mainly for Windows users, which usually don't like to use Terminal, so they could just start in this
simple python-like environment.

```
Tikmeh 0.1.0 (Sep 26, 2022) Sources and up-to-date executables: https://github.com/mehanon/tikmeh
Enter 'help' to get help message.
>>> directory mp4 tiktok.com/@shrimpydimpy/video/7133412834960018730
mp4/shrimpydimpy_2022-08-19_7133412834960018730.mp4
>>> profile @losertron
loading `@losertron` profile...
losertron/losertron_2022-09-14_7143063253696908586.mp4
losertron/losertron_2022-08-25_7135799462227660075.mp4
done.
>>> exit
```

### Building from sources

Note: requires Golang 1.19+

```shell
git clone https://github.com/mehanon/tikmeh
cd tikmeh
go build
```

### Caveats

1. The name is fucking retarded. Let's pretend it's
   after [Tikmeh (iranian village)](https://en.wikipedia.org/wiki/Tikmeh_Kord)
2. Windows anti-malware may not allow `tikmeh.exe` to access the internet, in this case administrator rights might help (idk how Windows work). 
You don't have to trust me, building from sources is always an option.  
3. Tikmeh depends on tikwm.com/api, which is the main bottleneck (1 request/10 sec is cringe)

### TODO:

- [ ] – become independent of tikwm to improve performance multiple times.
- [ ] – embed `ffmpeg` a way that don't require the user to download `ffmpeg` somewhere separately
  (is somewhat realised, but ffmpeg has to be provided by user)

#### Special thanks to [2ch.hk/media](https://2ch.hk/media) community for suggesting tikwm.com
