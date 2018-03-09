#!/usr/bin/env python

import plotly.offline as py
from plotly.graph_objs import *
import pandas as pd
import sys

LIMIT = 1000
SCALE = 5

mapbox_access_token = "YOUR_MAP_BOX_ACCESS_TOKEN"

if sys.version_info >= (3, 0):
    def xrange(arg, *rest):
        return range(arg, *rest)
    
df = pd.read_csv(sys.argv[1])
df = df.sort_values(by='pop').tail(LIMIT)

df['text'] = df['name'] + '<br>Population ' + (df['pop']).astype(str)
colors = ["rgb(0,116,217)","rgb(255,65,54)","rgb(133,20,75)","rgb(255,133,27)","lightgrey"]
cities = []
scale = SCALE

limits = []

if not 'group' in df:
    df['group'] = 0

for level in xrange(5):
    # print "level: ", level
    try:
        maxval = df.loc[df['group'] == level].sort_values(by='pop').head(1)['pop'].iloc[0]
        minval = df.loc[df['group'] == level].sort_values(by='pop').tail(1)['pop'].iloc[0]
        limits.append((minval, maxval))
    except IndexError:
        pass
    
# print "LIMITS: ", limits
colors.reverse()

for i in range(len(limits)):
    lim = limits[i]
    #df_sub = df[lim[0]:lim[1]]
    df_sub = df.loc[df['group'] == i]
    city = Scattermapbox(
        lon = df_sub['lon'],
        lat = df_sub['lat'],
        text = df_sub['text'],
        mode='markers',
        marker = dict(
            size = df_sub['pop']/scale,
            color = colors[i],
            sizemode = 'area'
        ),
        name = '{0} - {1}'.format(lim[0],lim[1]) )
    cities.append(city)

layout = dict(
        title = 'IP World Map<br>(Click legend to toggle traces)',
        showlegend = True,
        hovermode='closest',
        mapbox = dict(
            accesstoken=mapbox_access_token,
            bearing=0,
            center=dict(lat=20, lon=-170),
            pitch=0,
            zoom=1
        ),
    )

fig = dict( data=cities, layout=layout )
py.plot( fig, validate=False, filename='ip-world-map-mapbox.html' )

