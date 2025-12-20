# HNSW nolij Demo

Let's use the new HNSW feature in the nolij example app.

In the example directory of the repo add a new directory example/hnsw-demo/ with a README.md that tells the reader they'll need to use nolij in the example/hnsw-demo/ directory (via go run or `nolij` command). Then in the same README.md describe the HNSW feature we're exploring with some command line examples, and a link to PEOPLE.md, PLACES.md, SPORTS.md, HOBBIES.md, and CAREERS.md.

## PEOPLE.md

In the same directory create PEOPLE.md and add 10 sports personalities (include Serena and Venus Williams), 10 computer science adjacent people (include Ada Lovelace and Grace Hopper), 10 western political figures (include Thatcher, Reagan, Bush and Blair), 10 musicians, 10 artists, 10 actors/actresses, and 10 civil rights activists. Add a level 2 heading for each sublist and make each person a link to ./PEOPLE/<name>.md.

Once we have done this we will have a lot of links that don't resolve, so we'll create those markdown files next. In the markdown file for each person, we can add links to other people but it is not actually critical we do that... we'll just see how it goes. What is important is really that those markdown files say something factual about the person they are about, what the person is known for, date of birth if we know it, date of death if they're no longer with us. There really is no hard and fast rules about the format or consistency between the files, and in fact we want it to be a bit discontinuous because that shows the usefulness of searching the HNSW way!

## PLACES.md

In the same directory create PLACES.md and add 10 countries and 100 cities. Add a level 2 heading for each country and make each country a link to ./PLACES/<country>.md. In the markdown file for each country, add a level 2 heading for each city and make each city a link to ./PLACES/<country>/<city>.md. If we have cities that don't have a corresponding country, it doesn't matter, create the structure anyway. As with the people, we want it to be a bit discontinuous. In the documents about specific countries and cities we can add links, but it doesn't matter if we don't... see what way the dice falls.

## SPORTS.md, HOBBIES.md, CAREERS.md

You can probably guess the drill by now. Add 20 or so sports, 20 or so hobbies, 20 or so careers.

## Anything else to consider?

1. If you want to load in wikipedia data for anything here, by all means do so. If you decide to download the data in a tarball for example, don't keep it after this work is done. You'd only use it to flesh out the markdown files.

2. If you want to add more documents in other categories of things, by all means do so. The purpose of the demo is to show how embeddings can be used as a practical tool to heal disconnected knowledge in graph databases.

## Final Notes

Test. Test and test, and test some more. You may find bugs exploring this fun project.

Create as many tickets with the workitem tool as you wish if it assists you in organising (especially if autopilot is enabled!).
