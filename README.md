transcodequeue
==============

A utility to queue up handbrakecli re-encoding of files.  Used mostly to
transcode tv recoreded by EyeTV.

I have EyeTV and a tuner on an old mac.  It records tv off an antenna.  EyeTV
has scripts that you can register to be run on certain events, such as
RecordingDone and ExportDone.  I have a script that tells EyeTV to export a
file to a given location as a MPEGPS when its done recording it.  Further,
once its done Exporting, another script is fired off that I will use to
transcode the data to M4V.  I do this mostly to make the files smaller.

When transcribing the files I use HandbrakeCLI, and its possible that I will
save a show off, and transcode it while another is recording.  If the show was
long enough, or the computer slow enough, it might still be transcoding the
first show will still be transcoding when im done exporting the second one.
Rather than bring the machine to its knees trying to do two transcodes at
once, I want a solution that will queue up the transcode to be performed when
the first is done. 

This app is that solution.

If the app is run and no other instance of itself is running, it will fire up
a server, and ask that server to transcode a particular file and then when
done move it to the media server.  If the app is fired up and detects that its
already running it will communicate with the already running server and ask it
to do another transcode when the first is done.  Once all transcodes are done,
the app will exit.  

This was custom written for just my use and so the structure is pretty simple.
For instance, it knows its on a mac and asks the machine not to go to sleep
while transcoding (a problem I ran in to at one point).  It also just uses SCP
to copy the file over SSH to my media server.  Any attempts to make this more
useful would have to address these specific customizations, as well as others.


