-- Checks if element is a member of aset
function memberof(aset, element)
   if aset[element] ~= nil then
      return true
   else
      return false
   end
end

-- Makes the current item repeat on the specified weekdays (0 = sunday)
-- This function is supposed to be called in the !trigger column function
-- weekdays is a table with the desired weekdays as keys
--
-- example:
--    repeat_weekdays({ [0] = true, [1] = true })
-- makes the entry repeat on sunday and monday only
--
function repeat_weekdays(weekdays)
   next_when = when()
   if next_when <= 0 then
      return false
   end

   while true do
      next_when = next_when + (60 * 60 * 24)
      weekday = localtime(next_when).weekday
      if memberof(weekdays, weekday) then
	 break
      end
   end

   persist()
   clonecursor()
   when(next_when)
   
   return true
end