# Zurich API Package

https://opendatazurich.github.io/paris-api/

Get the 125 active members of the Zurich Gemeinderat and their functions. Removing members of the stadtrat and duplicates like Stimmenzählende, and Ratssekretariat.

```
curl -s -L "https://www.gemeinderat-zuerich.ch/api/behoerdenmandat/searchdetails?q=gremium%20adj%20Gemeinderat&l=de-CH" | python3 -c "
import sys
import xml.etree.ElementTree as ET

# Parse XML
tree = ET.parse(sys.stdin)
root = tree.getroot()

# Define namespaces
ns = {
    'bm': 'http://www.cmiag.ch/cdws/Behoerdenmandat',
    'k': 'http://www.cmiag.ch/cdws/Kontakt'
}

# Find all Behordenmandat elements with active end date
for mandat in root.findall('.//bm:Behordenmandat', ns):
    end = mandat.find('.//bm:End', ns)
    if end is not None and '9999-12-31' in end.text:
        name = mandat.find('.//bm:Name', ns)
        vorname = mandat.find('.//bm:Vorname', ns)
        funktion = mandat.find('.//bm:Funktion', ns)

        if name is not None and vorname is not None and funktion is not None:
            print(f'{vorname.text} {name.text} - {funktion.text}')
" | sort | grep -v "Stadtrat" | grep -v "Stimmenzählende" | grep -v "Ratssekretariat" | wc -l
     125
```
