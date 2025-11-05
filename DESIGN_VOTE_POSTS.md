# Design: X Posts for New Abstimmungen (Council Votes)

```
🗳️ Gemeinderat Zürich | Abstimmung vom 22.10.2025

✅ Angenommen: Postulat von @pascallamprecht (Grüne) und Ivo Bieri (SP) vom 11.09.2024: Schaffung eines zusätzlichen Treffpunkts im öffentlichen Raum in Witikon

📊 74 Ja | 41 Nein | 0 Enthaltung | 10 Abwesend

🔗 https://www.gemeinderat-zuerich.ch/geschaefte/2024-462
```

## Data Source

- [x] titel -> Abstimmung.Abstimmungstitel
- [x] datum -> Abstimmung.SitzungDatum
- [x] beschluss -> Abstimmung.Schlussresultat (map to ✅ Angenommen / ❌ Abgelehnt)
- [x] jaStimmen -> Abstimmung.JaStimmen
- [x] neinStimmen -> Abstimmung.NeinStimmen
- [x] enthalteneStimmen -> Abstimmung.EnthalteneStimmen
- [x] abwesendeMitglieder -> Abstimmung.AbwesendeMitglieder
- [x] for names check if we have a mapping to an X account and tag them (see `pkg/contacts/`)
- [ ] link -> Generate from Abstimmung.GeschaeftsID
