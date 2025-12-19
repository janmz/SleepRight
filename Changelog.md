# Changelog

Alle bemerkenswerten Änderungen an diesem Projekt werden in dieser Datei dokumentiert.

Das Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.0.0/),
und dieses Projekt folgt [Semantic Versioning](https://semver.org/lang/de/).

## [1.0.0] - 2025-01-12

### Hinzugefügt
- Initiale Version des SleepRight Tools
- Wake-Event Analyse aus Windows Event Log
- Anzeige aktueller Power-Einstellungen
- Konfiguration von Wake-Devices (nur Tastatur und Ethernet)
- Konfiguration von Sleep-Timeout (30 Minuten)
- Konfiguration von Power-Schema (Balanced)
- Konfigurierbares Hibernate-Timeout via `-wait` / `-w` Parameter
- CLI Interface mit Flags: `-info`, `-configure`, `-wait`, `-verbose`, `--version`
- Zweisprachige Dokumentation (Deutsch/Englisch)
- CI/CD Workflow für automatisiertes Testing und Build

### Funktionen
- `-info` / `-i`: Zeigt Wake-Events und aktuelle Power-Einstellungen
- `-configure` / `-c`: Konfiguriert Power-Einstellungen
- `-wait` / `-w <Minuten>`: Setzt Hibernate-Timeout in Minuten
- `-verbose` / `-v`: Ausführliche Ausgabe
- `--version`: Zeigt Version und Build-Zeit

[1.0.0]: https://github.com/janmz/SleepRight/releases/tag/v1.0.0

