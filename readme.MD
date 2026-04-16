# Transfer file via QRVideo

Convert a file to a QRCode video which can be captured via screen or camera. 

Processing the video back to original file.





## ⚡ Usage
- download index.html, open in browser.  drop and drop the file you want to transfer.
- Use screen capture or phone/camera to capture as video
- Run on windows, download qrvideo.exe and app-all.jar in same directory. install Java and set PATH to include java bin dir.

      qrvideo.exe <video_file>

- Run on other OS requirs build executable first

      go mod tidy
      go build qrvideo.go
      ...


## 📦 Prerequisites to run

- install Java and set PATH to include java bin





## 🚀 Build code
```bash

git clone https://github.com/yj703/Transfer_File_Via_QRVideo.git
cd Transfer_File_Via_QRVideo
go mod tidy
go build qrvideo.go


gradle build

```


## Note

- When capture screen,  you can install free tool OBS (https://obsproject.com/).  Windows snipping tool lost frame. 
- When capture with phone or camera,  put phone or camera on stable support or triport. Disable auto focus will help.
- Lower video speed (incrase delay to 200ms or 300ms) if video processing  failed.


