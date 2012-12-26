function tokremove(token) {
	$.ajax({ url: "/tokens?action=remove&token=" + encodeURIComponent(token),
		success: function(data, textStatus, req) {
			alert("Token removed");
		}});
}

function tokadd() {
	$.ajax({ url: "/tokens?action=add",
		success: function(data, textStatus, req) {
			alert("New token added");
		}});
}
