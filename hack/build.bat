REM Copyright 2019 The Skaffold Authors
REM
REM Licensed under the Apache License, Version 2.0 (the "License");
REM you may not use this file except in compliance with the License.
REM You may obtain a copy of the License at
REM
REM     http://www.apache.org/licenses/LICENSE-2.0
REM
REM Unless required by applicable law or agreed to in writing, software
REM distributed under the License is distributed on an "AS IS" BASIS,
REM WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
REM See the License for the specific language governing permissions and
REM limitations under the License.


REM A quick and dirty build file to build devloop on Windows.
REM Usage: hack\build.bat
REM
REM Disclaimer:
REM    This file is a good starting point for developing Devloop on a Windows machine.
REM    However, it might not be fully kept up to date with the Makefile -
REM    the Makefile is the source of truth for building devloop.

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1
FOR /F "tokens=*" %%a in ('git describe --always --tags --dirty') do SET VERSION=%%a
FOR /F "tokens=*" %%a in ('git rev-parse HEAD') do SET COMMIT=%%a
for /f %%a in ('powershell -Command "Get-Date -format yyyy_MM_dd__HH_mm_ss"') do set BUILD_DATE=%%a
set FLAG_LDFLAGS=" -X github.com/lucky-tools/devloop/pkg/devloop/version.version=%VERSION% -X github.com/lucky-tools/devloop/pkg/devloop/version.buildDate='%BUILD_DATE%' -X github.com/lucky-tools/devloop/pkg/devloop/version.gitCommit=%COMMIT% -extldflags \"\""

go build -ldflags %FLAG_LDFLAGS% -o out/devloop.exe github.com/lucky-tools/devloop/cmd/devloop
