## HWO 2014 Argusdusty Bot

This is a copy of the bot that set most of the records.
After building it with ./build, run it with ./bot testserver.helloworldopen.com 8091 imola > race.dat
Or run a multi-car race with ./bot hakkinen.helloworldopen.com 8091 suzuka 4 pwd botname > race.dat

If you're worried about performance, to get it to run in the real-time limit, you can lower the resolution on
DoThrottle (from 1<<5, 1<<3 to 1<<3, 1<<2, or something), and switch from DoTurbo2 (really slow), back to DoTurbo.

If you want to use it in competition mode, set that "if true" statement in main.go to "if false"


# Algorithms

All the algorithms are coded in logic.go, where the getNextPos function computes the next position of a single bot from a throttle value. Bumping is not supported, but switch physics are computed.

Position is calculated using a function that computes the next speed from the previous speed and the throttle, and two parameters, so:

    nextSpeed = (1.0-P.D)*P.X*throttle + P.D*speed

Where P.D is a drag parameter, and P.X is an engine power parameter. P.D is by default 0.98 (some people use P.D'=1.0-P.D = 0.02), and P.X is by default 10.0. In later competitions, P.D was constant.

Angle is calculated using a more complicated function, computing the next delta_angle from the previous delta_angle, the previous angle, the previous speed, the radius of the bend you are currently on, and four parameters, so:

    nextDeltaAngle = P.A*deltaAngle - P.S*speed*angle + sgn(curve.angle)*max((math.Sqrt(P.M*invrad)*speed - P.F) * speed, 0)

Where P.A is a momentum parameter, P.S is a normalization parameter, P.M is a slippery-ness parameter, and P.F is a grip parameter. P.A is by default 0.9 (some people use nextDeltaDeltaAngle = ..., and P.A -> -0.1), P.S is by default 0.00125 (1/800), P.M is by default 0.28125 (9/32), and P.F is by default 0.3. P.A was constant in all competitions, and P.S and P.F were proportional to each other (P.S = x/800, P.F=x*3/10)

Learning is done in learnbot.go, with the algorithms at the bottom, being exact solutions to the equations, just solving for the necessary variable parameters.

Switches had the most tricky physics. All switches are made up of 100 distinct radii, but let's start with their lengths. Switches on bends approximate a quadratic curve. This curve is defined by 3 points, The start, the end, and the midpoint, which the curve passes through. This is calculated via a bezier curve (where the P1 value is adjusted to get the curve to pass through the midpoint).

While quadratic bezier curves do have an exact expression for length, this is not used. Instead, the length is approximated (a little short), by calculating the straight-line-length between 10,000 points (100 points for each radii segment) along the bezier curve (equal-spaced t values). BendSwitchLength calculated this length. There is a bug in this code, where in actuality 10,001 points are used (with an extra piece on the end), which is why the switch going from opposite sides on a bend have slightly differing lengths (You may have noticed the difference is on the order of 1/10000 times their true length).

    curve = lambda t: quadBez((0, 0), 2*midPoint-endPoint/2, endPoint, t/10000)
    switchLength = sum(dist(curve(t+1), curve(t)) for t in range(10001))

Where endPoint/2 -> (endPoint.x/2, endPoint.y/2), and make sure to calculate it correctly starting from (0, 0), and midPoint is the angle/2, (startLane+endLane)/2 point. These are calculated using:

    x = startRadius - cos(angle*pi/180)*pointRadius
    y = -sin(angle*pi/180)*pointRadius

With midPoint having radius = (startRadius+endRadius)/2, and angle = bendAngle/2

For calculating the radii, the calculation tries to produce equal-length segments (as the radii are applied over equal-length segments). It does this using an "if partial_length > total_length/100", where partial length is calculated along those 10,000 points. It stores the previous two end-points (starting at 0, 0, which cause the first radii to always be NaN (which the server treats like a straight)), and uses those to calculate the radii of a circle fitting those three points. This is implemented in ApproxBendRadii (which is a misnomer, left over from when I was approximating it).

    prevPrevT = 0
    prevT = 0
    partialLength = 0
    radii = []
    for t in range(10001):
        partialLength += dist(curve(t+1), curve(t))
        if partialLength >= switchLength/100:
            partialLength -= switchLength/100
            radii.append(radiiFrom3points(curve(prevPrevT), curve(prevT), curve(t)))
            prevPrevT, prevT = prevT, t

(make sure to correct the first radii, and make sure you get the last radii, as sometimes at the end of this loop you may just have 99 elements in radii due to rounding errors)

Straight switches are calculated using the same techniques, but instead of a quadratic curve, it's a cubic curve (again, with a cubic bezier). There are four parameters, to determine the middle two points. These are ((0.1, 0.25), (0.875, 0.75)), and are proportional with the height/length of the switch. Slipping is disabled from straights, so no radii can be calculated (though they could be). The length is implemented in StraightSwitchLength.

    curve = lambda t: cubeBez((0, 0), (0.25*length, 0.1*laneDelta), (0.75*length, 0.875*laneDelta), (length, laneDelta), t/10000)
    straightSwitchLength = sum(dist(curve(t+1), curve(t)) for t in range(10001))


# Strategy

Throttle is implemented in speedbot.go, turbo in turbobot.go, switching in switchbot.go, and overtaking/bump prevention in oppbot.go

The throttle strategy is to pick two throttles to some resolution for the next two turns (currently 1/32 resolution for the first, 1/8 to the second), then simulate a greedy throttle out some turns (currently 25). Whichever gets us farthest is selected (using the first turn).

Turbo strategy was to use the longest straight(-ish) segment. I've upgraded it (DoTurbo2) to pick the turbo that gets us farthest by the time the next turbo is available (600 ticks between turbos).

Switch strategy is using shortest route (with a small switch cost of 5%). This is calculated efficiently using memoization techniques.

Overtaking is done by calculated an expected speed of each bot, and taking the switch which gets us to the next switch as fast as possible.

Bump avoidance is done by reducing maxAngle based on how likely (and how hard) the closest bot is to bumping me, with a small increase in their speed.