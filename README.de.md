# SleepRight

SleepRight ist ein Windows Sleep-Management-Tool, das dabei hilft, Windows 11 PCs für einen verlässlichen Schlafmodus zu konfigurieren und Sleep-bezogene Probleme zu beheben.

## Funktionen

- **Wake-Event Analyse**: Liest und zeigt Informationen über die letzten Wake-Events aus dem Windows Event Log an
- **Power Settings Übersicht**: Zeigt aktuelle Power-Schema-Einstellungen, Sleep-Timeouts, Wake-Device-Konfiguration und Hibernate-Einstellungen an
- **Power-Konfiguration**: Konfiguriert den PC so, dass nur Tastatur und Ethernet ihn aufwecken können, mit 30 Minuten Inaktivität bis zum Schlafmodus
- **Hibernate-Timeout**: Konfigurierbares Hibernate-Timeout über Kommandozeilen-Parameter

## Installation

### Aus Quellcode bauen

```bash
git clone https://github.com/janmz/SleepRight.git
cd SleepRight
go build
```

Oder verwenden Sie prepareBuild zum Bauen mit Versionsinformationen:

```bash
prepareBuild
```

## Verwendung

### Informationen anzeigen

Wake-Events und aktuelle Power-Einstellungen anzeigen:

```bash
SleepRight -info
# oder
SleepRight -i
```

### Power-Einstellungen konfigurieren

Power-Einstellungen mit Standardwerten konfigurieren (30 Minuten Sleep-Timeout, Balanced Power-Schema):

```bash
SleepRight -configure
# oder
SleepRight -c
```

### Konfiguration mit benutzerdefiniertem Hibernate-Timeout

Power-Einstellungen mit einem benutzerdefinierten Hibernate-Timeout (in Minuten) konfigurieren:

```bash
SleepRight -configure -wait 60
# oder
SleepRight -c -w 60
```

### Ausführliche Ausgabe

Ausführliche Ausgabe für detaillierte Informationen aktivieren:

```bash
SleepRight -info -verbose
# oder
SleepRight -i -v
```

### Version anzeigen

Version und Build-Zeit anzeigen:

```bash
SleepRight --version
```

## Kommandozeilen-Optionen

- `-info`, `-i` - Zeigt Wake-Events und aktuelle Power-Einstellungen an
- `-configure`, `-c` - Konfiguriert Power-Einstellungen
- `-wait`, `-w <Minuten>` - Setzt Hibernate-Timeout in Minuten (z.B. `-w 60` für 60 Minuten)
- `-verbose`, `-v` - Ausführliche Ausgabe
- `--version` - Zeigt Version und beendet das Programm

## Was SleepRight konfiguriert

Wenn Sie `SleepRight -configure` ausführen, wird folgendes konfiguriert:

1. **Wake-Devices**: Aktiviert Wake nur für Tastatur und Ethernet-Adapter, deaktiviert alle anderen Wake-Devices
2. **Sleep-Timeout**: Setzt Sleep-Timeout auf 30 Minuten Inaktivität (sowohl AC als auch Batterie)
3. **Power-Schema**: Setzt Power-Schema auf "Balanced"
4. **Hibernate-Timeout**: Konfiguriert Hibernate-Timeout, wenn `-wait` Parameter angegeben wird

## Anforderungen

- Windows 11 (funktioniert möglicherweise auch unter Windows 10)
- Administrator-Rechte erforderlich für Power-Konfiguration
- Go 1.23 oder höher (für das Bauen aus Quellcode)

## Beispiele

### Beispiel 1: Aktuelle Einstellungen prüfen

```bash
SleepRight -info
```

Ausgabe:
```
SleepRight v1.0.0.0 (Build: 2025-01-12 00:00:00)
=== Wake Events ===
Last Wake Event:
Wake History Count - 1
Wake History [0]
  Wake Source Count - 0

=== Power Settings ===
Active Power Scheme:
Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced)

Sleep Settings (AC Power):
  Timeout: 30 Minuten

Hibernate Settings:
  Timeout: 0 Sekunden (deaktiviert)

Geräte, die den Computer aufwecken können:
  - HID Keyboard Device
  - Realtek PCIe GbE Family Controller
```

### Beispiel 2: Konfiguration mit 60 Minuten Hibernate

```bash
SleepRight -configure -w 60
```

Dies konfiguriert alle Power-Einstellungen und setzt das Hibernate-Timeout auf 60 Minuten.

## Fehlerbehebung

### Administrator-Rechte erforderlich

Einige Operationen erfordern Administrator-Rechte. Wenn Sie Fehler bezüglich Berechtigungen sehen, führen Sie das Programm als Administrator aus:

1. Rechtsklick auf `SleepRight.exe`
2. "Als Administrator ausführen" wählen

### Wake-Devices nicht gefunden

Wenn SleepRight Ihre Tastatur oder Ihren Ethernet-Adapter nicht finden kann, können Sie Wake für spezifische Geräte manuell aktivieren:

```bash
powercfg /deviceenablewake "Gerätename"
```

Um alle verfügbaren Geräte zu sehen:

```bash
powercfg /devicequery wake_programmable
```

## Lizenz

MIT License mit Namensnennung. Donationware für CFI Kinderhilfe.

Copyright (c) 2024-2025 VAYA Consulting

## Beitragen

Beiträge sind willkommen! Bitte zögern Sie nicht, einen Pull Request einzureichen.

## Support

Für Probleme und Feature-Anfragen verwenden Sie bitte die [GitHub Issues](https://github.com/janmz/SleepRight/issues) Seite.

