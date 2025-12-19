# Changelog

Alle bemerkenswerten Änderungen an diesem Projekt werden in dieser Datei dokumentiert.

Das Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.0.0/),
und dieses Projekt folgt [Semantic Versioning](https://semver.org/lang/de/).

## [1.0.3.14] - 2025-12-19

### Verbessert
- Magic-Packet-Status: Wird jetzt explizit bei disabled Netzwerkgeräten angezeigt, um zu sehen, ob Magic-Packet aktiviert ist, auch wenn das Gerät selbst disabled ist

## [1.0.3.13] - 2025-12-19

### Behoben
- Exit-Code-Markierung: [EXIT_CODE:<nr>] wird jetzt aus der Ausgabe des elevated Child-Prozesses herausgefiltert und nicht mehr angezeigt
- WMI-Abfrage: Erweitert, um alle Netzwerkgeräte zu erfassen (nicht nur Active=True), um Magic-Packet-Status für alle Geräte anzuzeigen

### Verbessert
- Wake Device Ausgabe: Zuerst werden enabled Geräte angezeigt, dann disabled Geräte
- Magic-Packet-Status: Wird jetzt für alle Netzwerkgeräte angezeigt (Enabled/Disabled), nicht nur wenn aktiv
- Ausgabe-Struktur: Klarere Trennung zwischen enabled und disabled Geräten

## [1.0.3.12] - 2025-12-19

### Hinzugefügt
- WMI-Integration: Abfrage von Netzwerkkarten-Wake-Einstellungen über Windows Management Instrumentation
- WMI-Details: Zeigt an, ob Netzwerkkarten auf Magic-Packet-Only beschränkt sind
- UTF-8 zu Windows-Codepage Konvertierung: Funktionen `printUTF8` und `printUTF8ln` für korrekte Ausgabe von UTF-8 Zeichen in der Windows-Konsole
- Erweiterte Wake-Device-Details: Zeigt zusätzliche Informationen für enabled Geräte, insbesondere WMI-Daten für Netzwerkkarten

### Verbessert
- Wake Device Analyse: Detailliertere Ausgabe mit WMI-Informationen für Netzwerkkarten
- Zeichensatz-Handling: Alle UTF-8 Ausgaben werden automatisch in Windows-Codepage (CP1252) konvertiert

## [1.0.3.11] - 2025-12-19

### Behoben
- Event-Log-Parsing: Verbesserte Event-Block-Erkennung, um sicherzustellen, dass Sleep Time und Wake Time aus demselben Event stammen
- Zeitzone-Handling: UTC-Zeiten werden jetzt korrekt in die lokale Zeitzone konvertiert für die Anzeige
- Event-Parsing: Entfernung von unsichtbaren Zeichen (left-to-right mark) die das Parsen stören können
- Validierung: Prüfung, dass Wake Time nach Sleep Time liegt, bevor ein Event hinzugefügt wird

## [1.0.3.10] - 2025-12-19

### Verbessert
- Wake Device Analyse: Zeigt jetzt alle wake-programmable Geräte mit ihrem aktuellen Status (Enabled/Disabled)
- Wake Device Analyse: Übersichtliche Darstellung aller Geräte, die potentiell Wake-fähig sind, mit Status-Anzeige

## [1.0.3.9] - 2025-12-19

### Behoben
- Schlafdauer-Berechnung: Berechnet jetzt korrekt die Differenz zwischen "Zeit im Energiesparmodus" und "Reaktivierungszeit" im selben Event (nicht mehr zwischen verschiedenen Events)
- Event-Log-Analyse: Zeigt jetzt auch "Schlafbeginn" zusätzlich zur "Aufwachzeit" an

## [1.0.3.8] - 2025-12-19

### Verbessert
- Event Log Analyse: Events werden jetzt nach Datum absteigend sortiert (neueste zuerst)
- Event Log Analyse: Zeigt jetzt auch die Schlafdauer zwischen Wake Events in lesbarem Format (Tage/Stunden/Minuten/Sekunden)
- Event Log Analyse: Erhöhte Anzahl der abgerufenen Events von 10 auf 20 für bessere Sortierung

## [1.0.3.7] - 2025-12-19

### Behoben
- Zeichensatz-Problem: Ausgabe wird nicht mehr nach UTF-8 konvertiert, sondern bleibt in Windows-Codepage (CP1252) für korrekte Konsolen-Darstellung
- Power Settings Parsing: Erkennt jetzt deutsche Begriffe "Wechselstromeinstellung" (AC) und "Gleichstromeinstellung" (DC)
- Power Settings Parsing: Parst Hex-Werte direkt (z.B. 0x00003840 = 14400 Sekunden = 240 Minuten)

## [1.0.3.6] - 2025-12-19

### Behoben
- Power Requests Warning wird nicht mehr fälschlicherweise ausgegeben, wenn alle Werte "Keine" sind
- Power Settings Parsing verbessert: Sleep/Hibernate-Zeiten werden jetzt korrekt erkannt und angezeigt
- Event Log Ausgabe formatiert: Reaktivierungszeiten werden jetzt sauber mit nur Zeit und Quelle angezeigt

### Hinzugefügt
- `-debug` Flag: Zeigt alle externen Kommandoaufrufe mit vollständiger Ausgabe für Debugging-Zwecke

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

