on RecordingDone(recordingID)
	tell application "EyeTV"
		set new_recording_id to (recordingID as integer)
		set new_recording to recording id new_recording_id
		my export_recording(new_recording)
	end tell
end RecordingDone

on run
	tell application "EyeTV"
		set selected_recordings to selection of programs window
		repeat with selected_recording in selected_recordings
			my export_recording(selected_recording)
		end repeat
	end tell
end run

on export_recording(the_recording)
	tell application "EyeTV"
		set thisTitle to title of the_recording
		set cleanTitle to my escape_string(thisTitle)
		set thisEpisode to episode of the_recording
		set cleanEp to my escape_string(thisEpisode)
		export from the_recording to file ("Macintosh HD:Users:me:Downloads:" & cleanTitle & " - " & cleanEp & ".mpg") as MPEGPS with replacing without opening
	end tell
end export_recording

on escape_string(the_string)
	set clean_text to do shell script "echo " & quoted form of the_string & "| sed 's|[^a-zA-Z0-9 -.]||g' "
	return clean_text
end escape_string
