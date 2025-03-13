Skips updating the HTTPRoute if the weight is already equivalent to desired for all backend refs it should be changing.

Updated tests as well to include the new condition where things should not update.
