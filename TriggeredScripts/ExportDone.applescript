property HANDBRAKE_CLI : "/usr/local/bin/HandBrakeCLI"
property HANDBRAKE_PARAMETERS : " -O -I -f mp4 --encoder x264 --x264-preset faster --x264-tune film --h264-profile auto --h264-level auto --quality 20  --large-file --aencoder ca_aac,copy:ac3 -B 160 --mixdown dpl2 --aname English --loose-anamorphic --decomb --modulus 2"
property TARGET_PATH : "/Users/me/Downloads/"
property TARGET_TYPE : ".m4v"
property SOURCE_TYPE : ".mpg"
property SHELL_SCRIPT_SUFFIX : " >> /Users/me/Downloads/EyeTVExport.log 2>&1 & "

on run
	tell application "EyeTV"
		set selected_recordings to selection of programs window
		repeat with selected_recording in selected_recordings
			my transcode_recording(selected_recording)
			my delete_recording(selected_recording)
		end repeat
	end tell
end run

on ExportDone(recordingID)
	tell application "EyeTV"
		set new_recording_id to (recordingID as integer)
		set new_recording to recording id new_recording_id
		my transcode_recording(new_recording)
		my delete_recording(new_recording_id)
	end tell
end ExportDone

on delete_recording(the_recording)
	tell application "EyeTV"
		delete recording id the_recording
	end tell
end delete_recording

on transcode_recording(the_recording)
	tell application "EyeTV"
		set thisTitle to title of the_recording
		set cleanTitle to my clean_string(thisTitle)
		set thisEpisode to episode of the_recording
		set cleanEp to my clean_string(thisEpisode)
		set input_file to my escape_path(TARGET_PATH & cleanTitle & " - " & cleanEp & SOURCE_TYPE) as string
		set output_file to my escape_path(TARGET_PATH & cleanTitle & " - " & cleanEp & TARGET_TYPE) as string
		do shell script "nice ~/Projects/transcodequeue/bin/transcodequeue " & input_file & SHELL_SCRIPT_SUFFIX
	end tell
end transcode_recording

on escape_path(the_path)
	set oldDelimiters to AppleScript's text item delimiters
	set AppleScript's text item delimiters to "/"
	set path_components to every text item of the_path
	set AppleScript's text item delimiters to oldDelimiters
	repeat with counter from 1 to count path_components
		set path_component to item counter of path_components
		set item counter of path_components to my escape_string(path_component)
	end repeat
	set AppleScript's text item delimiters to "/"
	set the_path to path_components as string
	set AppleScript's text item delimiters to oldDelimiters
	return the_path
end escape_path

on clean_string(the_string)
	set clean_text to do shell script "echo " & quoted form of the_string & "| sed 's|[^a-zA-Z0-9 -.]||g' "
	return clean_text
end clean_string

on escape_string(input_string)
	
	set output_string to ""
	set escapable_characters to " !#^$%&*?()={}[]'`~|;<>\"\\"
	
	repeat with chr in input_string
		
		if (escapable_characters contains chr) then
			set output_string to output_string & "\\" -- This actually adds ONE \ to the string.
		else if (chr is equal to "/") then
			set output_string to output_string & ":" -- Swap file system delimiters
		end if
		
		set output_string to output_string & chr
		
	end repeat
	
	return output_string as text
	
end escape_string
