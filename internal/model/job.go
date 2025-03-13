package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QualityPreset represents predefined encoding quality profiles
type QualityPreset string

const (
	// PresetUltraFast prioritizes encoding speed over everything else.
	// Use for quick previews, testing, or when immediate results are needed.
	// - Minimal compression, very high bitrate requirements
	// - Visible artifacts at low bitrates
	// - 10-20x faster than balanced preset
	PresetUltraFast QualityPreset = "ultrafast"

	// PresetFast offers good encoding speed with reasonable quality.
	// Use for time-sensitive encoding where quality is still important.
	// - Moderate compression efficiency
	// - Good for live streaming or quick turnaround projects
	// - 2-3x faster than balanced with ~15% quality loss
	PresetFast QualityPreset = "fast"

	// PresetBalanced (DEFAULT) provides a good balance between speed and quality.
	// Recommended for most encoding scenarios.
	// - Good compression efficiency
	// - Reasonable encoding times
	// - Best choice for general-purpose encoding
	PresetBalanced QualityPreset = "balanced"

	// PresetQuality prioritizes visual quality over encoding speed.
	// Use when quality is more important than encoding time.
	// - Very good compression efficiency
	// - Slower encoding (2-3x slower than balanced)
	// - Better preservation of details in complex scenes
	PresetQuality QualityPreset = "quality"

	// PresetSlow significantly prioritizes quality over speed.
	// Use for final production versions where encoding time isn't critical.
	// - Excellent compression efficiency
	// - 3-5x slower than balanced preset
	// - Greatly improved detail preservation in complex scenes
	PresetSlow QualityPreset = "slow"

	// PresetUltraSlow maximizes compression efficiency and quality.
	// Use for archival masters or when encoding time doesn't matter.
	// - Maximum compression efficiency
	// - 8-15x slower than balanced preset
	// - Best possible quality with chosen codec
	PresetUltraSlow QualityPreset = "ultraslow"
)

// DefaultQualityPreset is the preset used when none is specified
const DefaultQualityPreset = PresetBalanced

// SimpleOptions provides an easy interface for novice users
type SimpleOptions struct {
	// Quality preset selection
	QualityPreset QualityPreset `json:"quality_preset,omitempty"`

	// Video options
	Resolution             string `json:"resolution,omitempty"` // "720p", "1080p", "4k", or "original"
	KeepOriginalResolution bool   `json:"keep_original_resolution,omitempty"`

	// Hardware acceleration
	UseHardwareAcceleration bool `json:"use_hardware_acceleration,omitempty"`

	// Trim video
	TrimFrom     string `json:"trim_from,omitempty"`     // e.g. "00:01:30"
	TrimDuration string `json:"trim_duration,omitempty"` // e.g. "00:10:00"

	// Audio options
	AudioQuality string `json:"audio_quality,omitempty"` // "low", "medium", "high"
}

// Job represents a transcoding job with complete FFmpeg argument control.
type Job struct {
	// Basic job properties
	InputFilePath       string `json:"input_file_path"`
	OutputFilePath      string `json:"output_file_path"`
	InputContainerType  string `json:"input_container_type,omitempty"`
	OutputContainerType string `json:"output_container_type,omitempty"`
	DryRun              string `json:"dry_run,omitempty"`

	// Simple options for novice users (used externally)
	SimpleOptions *SimpleOptions `json:"simple_options,omitempty"`

	// Advanced FFmpeg control (for power users)
	GlobalArguments string `json:"global_arguments,omitempty"`
	InputArguments  string `json:"input_arguments,omitempty"`
	OutputArguments string `json:"output_arguments,omitempty"`

	// Hardware device configuration (set by worker service)
	HardwareDevice string `json:"hardware_device,omitempty"`
}

// IsValidQualityPreset checks if the given preset is valid
func IsValidQualityPreset(preset QualityPreset) bool {
	switch preset {
	case PresetUltraFast, PresetFast, PresetBalanced,
		PresetQuality, PresetSlow, PresetUltraSlow:
		return true
	default:
		return false
	}
}

// GetPresetDescription returns a human-readable description of the preset
func GetPresetDescription(preset QualityPreset) string {
	switch preset {
	case PresetUltraFast:
		return "Maximum speed, lower quality (good for testing)"
	case PresetFast:
		return "Fast encoding with good quality"
	case PresetBalanced:
		return "Balanced speed and quality (recommended)"
	case PresetQuality:
		return "High quality, slower encoding"
	case PresetSlow:
		return "Very high quality, slow encoding"
	case PresetUltraSlow:
		return "Maximum quality, extremely slow encoding"
	default:
		return "Unknown preset"
	}
}

// GetFFmpegPresetArgs returns FFmpeg arguments for the given quality preset and codec
func GetFFmpegPresetArgs(preset QualityPreset, useHardwareAccel bool) string {
	// Default to balanced if preset is invalid
	if !IsValidQualityPreset(preset) {
		preset = DefaultQualityPreset
	}

	// For hardware acceleration
	if useHardwareAccel {
		switch preset {
		case PresetUltraFast:
			return "-c:v av1_qsv -preset veryfast -look_ahead_depth 8"
		case PresetFast:
			return "-c:v av1_qsv -preset faster -look_ahead_depth 16"
		case PresetQuality:
			return "-c:v av1_qsv -preset slow -look_ahead_depth 40"
		case PresetSlow:
			return "-c:v av1_qsv -preset slower -look_ahead_depth 60"
		case PresetUltraSlow:
			return "-c:v av1_qsv -preset veryslow -look_ahead_depth 120"
		default: // Balanced
			return "-c:v av1_qsv -preset slow -look_ahead_depth 30"
		}
	}

	// For software encoding using AV1
	switch preset {
	case PresetUltraFast:
		return "-c:v libaom-av1 -crf 30 -b:v 0 -cpu-used 8 -row-mt 1"
	case PresetFast:
		return "-c:v libaom-av1 -crf 30 -b:v 0 -cpu-used 6 -row-mt 1"
	case PresetQuality:
		return "-c:v libaom-av1 -crf 28 -b:v 0 -cpu-used 2 -row-mt 1"
	case PresetSlow:
		return "-c:v libaom-av1 -crf 25 -b:v 0 -cpu-used 1 -row-mt 1"
	case PresetUltraSlow, PresetHighQuality:
		return "-c:v libaom-av1 -crf 20 -b:v 0 -cpu-used 0 -row-mt 1 -tiles 2x2"
	default: // Balanced
		return "-c:v libaom-av1 -crf 30 -b:v 0 -cpu-used 4 -row-mt 1"
	}
}

// UnmarshalJSON provides custom unmarshaling logic
func (j *Job) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid recursion in UnmarshalJSON
	type JobAlias Job
	aux := (*JobAlias)(j)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Set default preset if not specified
	if j.SimpleOptions != nil && j.SimpleOptions.QualityPreset == "" {
		j.SimpleOptions.QualityPreset = DefaultQualityPreset
	}

	// If SimpleOptions provided but no advanced arguments, generate them
	if j.SimpleOptions != nil && j.GlobalArguments == "" &&
		j.InputArguments == "" && j.OutputArguments == "" {
		// Convert simple options to FFmpeg arguments
		j.convertSimpleOptionsToArguments()
	}

	return nil
}

// IsAdvancedMode returns whether the job is using advanced FFmpeg argument control
func (j *Job) IsAdvancedMode() bool {
	return j.GlobalArguments != "" || j.InputArguments != "" || j.OutputArguments != ""
}

// IsDryRun checks if this is a dry run job
func (j *Job) IsDryRun() bool {
	return strings.ToLower(j.DryRun) == "true"
}

// convertSimpleOptionsToArguments translates user-friendly options to FFmpeg arguments
func (j *Job) convertSimpleOptionsToArguments() {
	opts := j.SimpleOptions
	if opts == nil {
		return
	}

	// Set reasonable global arguments
	j.GlobalArguments = "-y -hide_banner"

	// Handle trimming (these are input arguments)
	var inputArgs []string
	if opts.TrimFrom != "" {
		inputArgs = append(inputArgs, fmt.Sprintf("-ss %s", opts.TrimFrom))
		if opts.TrimDuration != "" {
			inputArgs = append(inputArgs, fmt.Sprintf("-t %s", opts.TrimDuration))
		}
	}
	j.InputArguments = strings.Join(inputArgs, " ")

	// Handle output arguments (video and audio encoding)
	var outputArgs []string

	// Use the new helper function for preset arguments
	presetArgs := GetFFmpegPresetArgs(opts.QualityPreset, opts.UseHardwareAcceleration)
	outputArgs = append(outputArgs, presetArgs)

	// Handle resolution
	if !opts.KeepOriginalResolution && opts.Resolution != "" &&
		opts.Resolution != "original" {
		// Convert common terms to actual dimensions
		var dimension string
		switch strings.ToLower(opts.Resolution) {
		case "480p":
			dimension = "854:480"
		case "720p":
			dimension = "1280:720"
		case "1080p":
			dimension = "1920:1080"
		case "4k", "2160p":
			dimension = "3840:2160"
		default:
			dimension = opts.Resolution // Assume it's already in the right format
		}

		outputArgs = append(outputArgs, fmt.Sprintf("-vf scale=%s", dimension))
	}

	// Handle audio quality
	switch strings.ToLower(opts.AudioQuality) {
	case "low":
		outputArgs = append(outputArgs, "-c:a libopus -b:a 64k")
	case "high":
		outputArgs = append(outputArgs, "-c:a libopus -b:a 256k")
	default: // medium or unspecified
		outputArgs = append(outputArgs, "-c:a libopus -b:a 128k")
	}

	j.OutputArguments = strings.Join(outputArgs, " ")
}

// GetFFmpegCommand generates the complete FFmpeg command for this job
func (j *Job) GetFFmpegCommand() []string {
	var args []string

	args = j.addHardwareDeviceArgs(args)
	args = j.addGlobalArgs(args)
	args = j.addInputArgs(args)
	args = j.addInputFile(args)
	args = j.addOutputArgs(args)
	args = j.addOutputFile(args)

	return args
}

func (j *Job) addOutputFile(args []string) []string {
	return append(args, j.OutputFilePath)
}

func (j *Job) addHardwareDeviceArgs(args []string) []string {
	if j.HardwareDevice != "" {
		args = append(args, "-init_hw_device", j.HardwareDevice)
	} else if j.SimpleOptions != nil && j.SimpleOptions.UseHardwareAcceleration {
		args = append(args, "-init_hw_device", "vaapi=va:/dev/dri/renderD128")
	}
	return args
}

func (j *Job) addGlobalArgs(args []string) []string {
	// Add any explicitly provided global arguments first
	if j.GlobalArguments != "" {
		for _, arg := range strings.Fields(j.GlobalArguments) {
			args = append(args, arg)
		}
	}

	// Apply standard global args if none specified
	if j.GlobalArguments == "" {
		args = append(args, "-y", "-hide_banner")
	}

	return args
}

func (j *Job) addInputArgs(args []string) []string {
	if j.InputArguments != "" {
		for _, arg := range strings.Fields(j.InputArguments) {
			args = append(args, arg)
		}
	}
	return args
}

func (j *Job) addInputFile(args []string) []string {
	return append(args, "-i", j.InputFilePath)
}

func (j *Job) addOutputArgs(args []string) []string {
	if j.SimpleOptions != nil {
		presetArgs := GetFFmpegPresetArgs(j.SimpleOptions.QualityPreset, j.SimpleOptions.UseHardwareAcceleration)
		args = append(args, strings.Fields(presetArgs)...)
		args = append(args, "-c:a", "libopus", "-b:a", "256k")
	} else {
		// Default fallback if SimpleOptions is nil
		args = append(args, strings.Fields(GetFFmpegPresetArgs(DefaultQualityPreset, j.SimpleOptions.UseHardwareAcceleration))...)
	}
}
