# AstroSession Creator

AstroSession Creator is an ultra-fast, standalone CLI tool written in Go that automates the creation of folder structures for astrophotography sessions. It intelligently queries the CDS Sesame/SIMBAD API to resolve common and technical names for celestial objects, preventing duplicate captures and organizing your post-processing pipeline.

## Features
- **Intelligent Naming**: Automatically queries SIMBAD for common conversational names (e.g. converting `M81 M82` into `M81_M82 (Bode's & Cigar Galaxies)`).
- **Concurrent File Mover**: Quickly transfers gigabytes of your Flat and Light frames directly into their target directories using multi-threaded goroutines, with real-time ETA progress bars.
- **Stand-Alone**: Zero dependencies. No Python, no `astroquery` pip modules. Just a single executable file you can run natively on macOS, Linux, or Windows.
- **Duplicate Safety**: Automatically detects previously existing sessions and cleanly appends suffixes to avoid data overwrite.

## Folder Structure Output
It automatically creates the optimal storage topology for PixInsight workflows:
```
M81_M82 (Bode's & Cigar Galaxies)/
└── 2026/
    └── 22 feb/
        ├── Lights/
        │   └── Rejected/
        ├── Flats/
        ├── Logs/
        ├── PixInsight/
        └── Final/
```

## Download & Installation
You do NOT need to install Go to use this tool!
1. Go to the [Releases](https://github.com/coderGo93/AstroSession-Creator/releases) page on this repository.
2. Download the binary that matches your operating system:
   - **macOS** (`AstroSession-Creator_mac_intel` for Intel and Rosetta on Silicon Macs)
   - **Windows** (`AstroSession-Creator_windows_x64.exe`)
   - **Linux** (`AstroSession-Creator_linux_x64`)
3. Place the executable file in the root directory where you want your astrophotography objects to be generated.
4. **Mac & Linux Users Only:** Open your terminal, navigate to that folder, and grant execution permissions:
   ```bash
   chmod +x AstroSession-Creator_mac_intel
   ```
5. Run the tool from the terminal: `./AstroSession-Creator_mac_intel` (or double click it on Windows).

> **Note for macOS Users:**
> Since this open-source tool isn't signed with a paid Apple Developer certificate, macOS Gatekeeper might block it saying *"Apple could not verify [App] is free of malware"*.
> **To allow it:** Open your terminal and run `xattr -d com.apple.quarantine AstroSession-Creator_mac_intel`, or go to **System Settings > Privacy & Security** and click **"Allow Anyway"**.

## Development
If you wish to modify the search algorithm or folder structures:
```bash
git clone https://github.com/coderGo93/AstroSession-Creator.git
cd AstroSession-Creator
go run .
```

*Note: Binaries are automatically generated via GitHub Actions on every push to the `main` branch.*
