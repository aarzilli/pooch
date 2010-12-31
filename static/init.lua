function titleq(title_value)
   return { query = "title_query";
	    value = title_value }
end

function textq(text_value)
   return { query = "text_query";
	    value = text_value }
end

function whenq(operator, value)
   return { op = "when_query";
	    operator = operator;
	    value = value }
end

function idq(value)
   return { query = "id_query";
	    value = value }
end
