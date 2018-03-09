#!/usr/bin/env python

import plotly.offline as py
import pandas as pd
import sys

LIMIT = 1000
SCALE = 10

if sys.version_info >= (3, 0):
    def xrange(arg, *rest):
        return range(arg, *rest)
    
df = pd.read_csv(sys.argv[1], quotechar='"')

df['text'] = df['name'] + '<br>Population ' + (df['pop']).astype(str)
limits = [(0,2),(3,10),(11,20),(21,50),(50,3000)]
colors = ["rgb(0,116,217)","rgb(255,65,54)","rgb(133,20,75)","rgb(255,133,27)","lightgrey"]
cities = []
scale = SCALE

limits = []

if not 'group' in df:
    df['group'] = 0

for level in xrange(5):
    # print ("level: ", level)
    try:
        maxval = df.loc[df['group'] == level].sort_values(by='pop').head(1)['pop'].iloc[0]
        minval = df.loc[df['group'] == level].sort_values(by='pop').tail(1)['pop'].iloc[0]
        limits.append((minval, maxval))
    except IndexError:
        pass
    
# print ("LIMITS: ", limits)
colors.reverse()

for i in range(len(limits)):
    lim = limits[i]
    df_sub = df.loc[df['group'] == i]
    city = dict(
        type = 'scattergeo',
        locationmode = 'ISO-3',
        lon = df_sub['lon'],
        lat = df_sub['lat'],
        text = df_sub['text'],
        marker = dict(
            size = df_sub['pop']/scale,
            color = colors[i],
            line = dict(width=0.5, color='rgb(40,40,40)'),
            sizemode = 'area'
        ),
        name = '{0} - {1}'.format(lim[0],lim[1]) )
    cities.append(city)

layout = dict(
        title = 'Top %d IP World Map<br>(Click legend to toggle traces)' % LIMIT,
        showlegend = True,
        geo = dict(
            scope='world',
            projection=dict( type='natural earth' ),
            showland = True,
            landcolor = 'rgb(217, 217, 217)',
            subunitwidth=1,
            countrywidth=1,
            subunitcolor="rgb(255, 255, 255)",
            countrycolor="rgb(255, 255, 255)"
        ),
    )

fig = dict( data=cities, layout=layout )
py.plot( fig, validate=False, filename='ip-world-map-d3.html' )

