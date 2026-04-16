# Transfer file via QRVideo

Convert a file to a QRCode video which can be captured via screen capture tool or by camera/phone video recording.

Processing the recorded video file back to original file.





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

- install Java (https://www.java.com/en/download/) and set PATH to include java bin





## 🚀 Build code

You don't need run build to use this tool. only do if you have done some code work or port to different platform. 

To buld qrvideo executable binary

```bash
git clone https://github.com/yj703/Transfer_File_Via_QRVideo.git
cd Transfer_File_Via_QRVideo
go mod tidy
go build qrvideo.go
```

To build jar from java code

```
gradle build
```


## Note

- When capture screen,  you can install free tool OBS (https://obsproject.com/).  Windows snipping tool losts frame, doesn't work well.
- When capture with phone or camera,  put phone or camera on stable support or triport. Sometime disable phone auto focus improve video quality.
- Lower video speed (incrase delay to 200ms or 300ms) if video processing  failed.


